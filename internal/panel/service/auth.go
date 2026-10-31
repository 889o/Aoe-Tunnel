package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"aoe-tunnel/internal/panel/auth"
	"aoe-tunnel/internal/panel/model"
)

type TokenResult struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

type RoleBrief struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type MeResponse struct {
	User        model.User `json:"user"`
	Role        RoleBrief  `json:"role"`
	Permissions []string   `json:"permissions"`
	Menus       []MenuItem `json:"menus"`
}

func (s *Service) Login(ctx context.Context, username, password, ua, ip string) (*TokenResult, *Principal, error) {
	var user model.User
	err := s.db.WithContext(ctx).Where("username = ?", strings.TrimSpace(username)).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 用户名不存在与密码错误返回同一错误，避免探测账号。
			return nil, nil, ErrUnauthorized
		}
		return nil, nil, err
	}
	if !verifyPassword(user.PasswordHash, password) {
		return nil, nil, ErrUnauthorized
	}
	if user.Status != "active" {
		if user.Status == model.UserStatusPending {
			return nil, nil, fmt.Errorf("%w: account pending review", ErrForbidden)
		}
		return nil, nil, fmt.Errorf("%w: user disabled", ErrForbidden)
	}
	role, err := s.roleByID(ctx, user.RoleID)
	if err != nil {
		return nil, nil, err
	}

	sid, err := auth.NewSID()
	if err != nil {
		return nil, nil, err
	}
	refresh, err := s.tokens.SignRefresh(user.ID, sid)
	if err != nil {
		return nil, nil, err
	}
	access, err := s.tokens.SignAccess(user.ID, sid, role.Code)
	if err != nil {
		return nil, nil, err
	}
	if err := s.createSession(ctx, user.ID, sid, auth.HashToken(refresh), s.tokens.RefreshTTL, ua, ip); err != nil {
		return nil, nil, err
	}

	now := time.Now()
	if err := s.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", user.ID).
		Update("last_login_at", now).Error; err != nil {
		s.log.Warn("login: update last_login_at failed", zap.Error(err))
	}
	user.LastLoginAt = &now

	p := newPrincipal(user, *role, "jwt")
	p.SID = sid
	return &TokenResult{
		AccessToken:  access,
		RefreshToken: refresh,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.tokens.AccessTTL.Seconds()),
	}, p, nil
}

// Refresh 轮换 refresh token；旧 refresh token 立即失效，重复使用会被拒绝。
func (s *Service) Refresh(ctx context.Context, refreshToken string) (*TokenResult, *Principal, error) {
	claims, err := s.tokens.Parse(refreshToken, auth.TokenTypeRefresh)
	if err != nil {
		return nil, nil, ErrUnauthorized
	}
	newRefresh, err := s.tokens.SignRefresh(claims.UID, claims.SID)
	if err != nil {
		return nil, nil, err
	}
	rotated, err := s.rotateSession(ctx, claims.SID, auth.HashToken(refreshToken), auth.HashToken(newRefresh))
	if err != nil {
		return nil, nil, err
	}
	if !rotated {
		return nil, nil, ErrUnauthorized
	}

	p, err := s.LoadPrincipal(ctx, claims.UID, "jwt")
	if err != nil {
		return nil, nil, ErrUnauthorized
	}
	access, err := s.tokens.SignAccess(p.User.ID, claims.SID, p.Role.Code)
	if err != nil {
		return nil, nil, err
	}
	p.SID = claims.SID
	return &TokenResult{
		AccessToken:  access,
		RefreshToken: newRefresh,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.tokens.AccessTTL.Seconds()),
	}, p, nil
}

// Register 开放注册：是否允许注册与是否需要审核由站点配置决定。
func (s *Service) Register(ctx context.Context, username, password, inviteCode string) (*model.User, error) {
	cfg, err := s.SiteConfig(ctx)
	if err != nil {
		return nil, err
	}
	if !cfg.RegisterOpen {
		return nil, fmt.Errorf("%w: registration is closed", ErrForbidden)
	}
	username = strings.TrimSpace(username)
	if len(username) < 3 || len(username) > 64 {
		return nil, fmt.Errorf("username must be 3-64 characters")
	}
	if strings.ContainsAny(username, " \t\r\n") {
		return nil, fmt.Errorf("username must not contain whitespace")
	}
	role, err := s.roleByCode(ctx, RoleUser)
	if err != nil {
		return nil, err
	}
	var count int64
	if err := s.db.WithContext(ctx).Model(&model.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, fmt.Errorf("%w: username %s already exists", ErrConflict, username)
	}
	hash, err := hashPassword(password)
	if err != nil {
		return nil, err
	}
	status := "active"
	if cfg.RegisterReview {
		status = model.UserStatusPending
	}
	// 开启分销且邀请码有效时绑定上级；无效邀请码静默忽略，不影响注册本身。
	var inviterID *uint
	if cfg.DistributionOpen {
		if code := strings.TrimSpace(inviteCode); code != "" {
			var inviter model.User
			if err := s.db.WithContext(ctx).
				Where("invite_code = ? AND status = ?", code, "active").
				First(&inviter).Error; err == nil {
				id := inviter.ID
				inviterID = &id
			}
		}
	}
	u := model.User{
		Username:     username,
		PasswordHash: hash,
		Role:         role.Code,
		RoleID:       role.ID,
		Status:       status,
		InviterID:    inviterID,
	}
	// 注册即发放邀请码，注册成功后可在个人设置里直接分享。
	for attempt := 0; attempt < 5; attempt++ {
		code, err := randomInviteCode()
		if err != nil {
			return nil, err
		}
		var used int64
		if err := s.db.WithContext(ctx).Model(&model.User{}).Where("invite_code = ?", code).Count(&used).Error; err != nil {
			return nil, err
		}
		if used == 0 {
			u.InviteCode = code
			break
		}
	}
	if err := s.db.WithContext(ctx).Create(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Service) Logout(ctx context.Context, sid string) error {
	if sid == "" {
		return nil
	}
	return s.revokeSession(ctx, sid)
}

func (s *Service) Me(p *Principal) *MeResponse {
	return &MeResponse{
		User:        p.User,
		Role:        RoleBrief{Code: p.Role.Code, Name: p.Role.Name},
		Permissions: p.Permissions,
		Menus:       MenusFor(p.PermSet, p.Role.Code),
	}
}

func hashPassword(pw string) (string, error) {
	return auth.HashPassword(pw)
}

func verifyPassword(hash, pw string) bool {
	return auth.VerifyPassword(hash, pw)
}
