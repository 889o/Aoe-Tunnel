package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"aoe-tunnel/internal/panel/model"
)

// Principal 是请求身份：来自 JWT 或节点令牌，权限始终按所属角色实时加载。
type Principal struct {
	User        model.User
	Role        model.Role
	Permissions []string
	PermSet     map[string]struct{}
	Via         string
	SID         string
	// NodeCode/NodeType 仅在 X-Node-Token 通道填充，用于限定节点上线接口。
	NodeCode string
	NodeType int
}

func (p *Principal) Has(code string) bool {
	if p == nil {
		return false
	}
	_, ok := p.PermSet[code]
	return ok
}

func newPrincipal(user model.User, role model.Role, via string) *Principal {
	codes := role.PermissionCodes
	if codes == nil {
		codes = []string{}
	}
	return &Principal{User: user, Role: role, Permissions: codes, PermSet: PermissionSet(codes), Via: via}
}

// LoadPrincipal 按用户实时加载角色与权限；用户被禁用或角色被删除都会直接失败。
func (s *Service) LoadPrincipal(ctx context.Context, userID uint, via string) (*Principal, error) {
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if user.Status != "active" {
		return nil, fmt.Errorf("%w: user disabled", ErrForbidden)
	}
	role, err := s.roleByID(ctx, user.RoleID)
	if err != nil {
		return nil, err
	}
	return newPrincipal(user, *role, via), nil
}

type UserView struct {
	model.User
	RoleCode string `json:"role_code"`
	RoleName string `json:"role_name"`
}

func (s *Service) ListUsers(ctx context.Context) ([]UserView, error) {
	var users []model.User
	if err := s.db.WithContext(ctx).Order("id asc").Find(&users).Error; err != nil {
		return nil, err
	}
	roles, err := s.ListRoles(ctx)
	if err != nil {
		return nil, err
	}
	byID := make(map[uint]RoleView, len(roles))
	for _, r := range roles {
		byID[r.ID] = r
	}
	out := make([]UserView, 0, len(users))
	for _, u := range users {
		rv := byID[u.RoleID]
		out = append(out, UserView{User: u, RoleCode: rv.Code, RoleName: rv.Name})
	}
	return out, nil
}

type CreateUserInput struct {
	Username    string `json:"username" binding:"required,max=64"`
	Password    string `json:"password" binding:"required,min=6,max=72"`
	DisplayName string `json:"display_name,omitempty" binding:"max=64"`
	RoleCode    string `json:"role_code" binding:"required,max=32"`
	// PackageID 绑定套餐，创建端口规则时自动使用；0 或留空表示未绑定。
	PackageID *uint  `json:"package_id,omitempty"`
	Status    string `json:"status,omitempty"`
}

func (s *Service) CreateUser(ctx context.Context, in CreateUserInput) (*model.User, error) {
	username := strings.TrimSpace(in.Username)
	if username == "" {
		return nil, fmt.Errorf("username required")
	}
	role, err := s.roleByCode(ctx, strings.TrimSpace(in.RoleCode))
	if err != nil {
		return nil, fmt.Errorf("role_code: %w", err)
	}
	// 面板只暴露管理员/用户两种角色，系统角色不允许通过接口创建。
	if role.Code != RoleAdmin && role.Code != RoleUser {
		return nil, fmt.Errorf("%w: role must be admin or user", ErrForbidden)
	}
	var count int64
	if err := s.db.WithContext(ctx).Model(&model.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, fmt.Errorf("%w: username %s already exists", ErrConflict, username)
	}
	hash, err := hashPassword(in.Password)
	if err != nil {
		return nil, err
	}
	packageID, err := s.resolveUserPackage(ctx, in.PackageID)
	if err != nil {
		return nil, err
	}
	status := strings.TrimSpace(in.Status)
	if status == "" {
		status = "active"
	}
	if !validUserStatus(status) {
		return nil, fmt.Errorf("status must be active, pending or disabled")
	}
	u := model.User{
		Username:     username,
		PasswordHash: hash,
		DisplayName:  strings.TrimSpace(in.DisplayName),
		Role:         role.Code,
		RoleID:       role.ID,
		PackageID:    packageID,
		Status:       status,
	}
	if err := s.db.WithContext(ctx).Create(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

type UpdateUserInput struct {
	DisplayName *string `json:"display_name,omitempty" binding:"omitempty,max=64"`
	RoleCode    *string `json:"role_code,omitempty" binding:"omitempty,max=32"`
	Status      *string `json:"status,omitempty" binding:"omitempty,max=16"`
	Password    *string `json:"password,omitempty" binding:"omitempty,min=6,max=72"`
	// PackageID 0 表示解绑套餐，留空表示不修改绑定关系。
	PackageID *uint `json:"package_id,omitempty"`
}

func (s *Service) UpdateUser(ctx context.Context, id uint, in UpdateUserInput, actor *Principal) (*model.User, error) {
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if err := s.checkUserManageable(ctx, &user, actor); err != nil {
		return nil, err
	}

	if in.DisplayName != nil {
		user.DisplayName = strings.TrimSpace(*in.DisplayName)
	}
	if in.RoleCode != nil {
		if actor.Role.Code != RoleSuperAdmin && actor.Role.Code != RoleAdmin {
			return nil, fmt.Errorf("%w: only admin can assign roles", ErrForbidden)
		}
		if user.ID == actor.User.ID {
			return nil, fmt.Errorf("%w: cannot change own role", ErrForbidden)
		}
		roleCode := strings.TrimSpace(*in.RoleCode)
		// 非超级管理员只能在“管理员/用户”之间切换，避免提权到系统角色。
		if actor.Role.Code != RoleSuperAdmin && roleCode != RoleAdmin && roleCode != RoleUser {
			return nil, fmt.Errorf("%w: role must be admin or user", ErrForbidden)
		}
		role, err := s.roleByCode(ctx, roleCode)
		if err != nil {
			return nil, fmt.Errorf("role_code: %w", err)
		}
		user.RoleID = role.ID
		user.Role = role.Code
	}
	if in.Status != nil {
		status := strings.TrimSpace(*in.Status)
		if !validUserStatus(status) {
			return nil, fmt.Errorf("status must be active, pending or disabled")
		}
		if user.ID == actor.User.ID && status != "active" {
			return nil, fmt.Errorf("%w: cannot disable self", ErrForbidden)
		}
		user.Status = status
	}
	if in.PackageID != nil {
		packageID, err := s.resolveUserPackage(ctx, in.PackageID)
		if err != nil {
			return nil, err
		}
		user.PackageID = packageID
	}
	if in.Password != nil {
		hash, err := hashPassword(*in.Password)
		if err != nil {
			return nil, err
		}
		user.PasswordHash = hash
		// 改密后强制所有会话失效，避免旧 token 继续使用。
		if _, err := s.revokeAllSessions(ctx, user.ID); err != nil {
			return nil, err
		}
	}
	if user.Status == "disabled" {
		if _, err := s.revokeAllSessions(ctx, user.ID); err != nil {
			return nil, err
		}
	}
	if err := s.db.WithContext(ctx).Save(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Service) DeleteUser(ctx context.Context, id uint, actor *Principal) error {
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	if user.ID == actor.User.ID {
		return fmt.Errorf("%w: cannot delete self", ErrForbidden)
	}
	if err := s.checkUserManageable(ctx, &user, actor); err != nil {
		return err
	}
	if _, err := s.revokeAllSessions(ctx, user.ID); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Delete(&user).Error
}

// validUserStatus 允许的值：正常、待审核、禁用。
func validUserStatus(status string) bool {
	return status == "active" || status == model.UserStatusPending || status == "disabled"
}

// checkUserManageable 阻止低权限角色操作超级管理员账号。
func (s *Service) checkUserManageable(ctx context.Context, target *model.User, actor *Principal) error {
	if actor.Role.Code == RoleSuperAdmin {
		return nil
	}
	targetRole, err := s.roleByID(ctx, target.RoleID)
	if err != nil {
		return err
	}
	if targetRole.Code == RoleSuperAdmin {
		return fmt.Errorf("%w: cannot manage super_admin", ErrForbidden)
	}
	return nil
}

// resolveUserPackage 校验并归一化套餐绑定：0 表示解绑，非 0 必须存在。
func (s *Service) resolveUserPackage(ctx context.Context, id *uint) (*uint, error) {
	if id == nil || *id == 0 {
		return nil, nil
	}
	if _, err := s.getPackage(ctx, *id); err != nil {
		return nil, fmt.Errorf("package_id: %w", err)
	}
	v := *id
	return &v, nil
}

// Profile 返回个人设置所需的基本信息。
func (s *Service) Profile(ctx context.Context, userID uint) (*UserView, *model.Role, error) {
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrNotFound
		}
		return nil, nil, err
	}
	role, err := s.roleByID(ctx, user.RoleID)
	if err != nil {
		return nil, nil, err
	}
	return &UserView{User: user, RoleCode: role.Code, RoleName: role.Name}, role, nil
}

// UpdateProfile 允许用户修改自己的显示名与密码，不允许改角色与状态。
func (s *Service) UpdateProfile(ctx context.Context, userID uint, displayName, oldPassword, newPassword *string) (*model.User, error) {
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if displayName != nil {
		user.DisplayName = strings.TrimSpace(*displayName)
	}
	if newPassword != nil {
		if oldPassword == nil || !verifyPassword(user.PasswordHash, *oldPassword) {
			return nil, fmt.Errorf("%w: old password mismatch", ErrForbidden)
		}
		hash, err := hashPassword(*newPassword)
		if err != nil {
			return nil, err
		}
		user.PasswordHash = hash
		// 改密后统一撤销全部会话，写黑名单保证旧 access token 立即失效。
		if _, err := s.revokeAllSessions(ctx, user.ID); err != nil {
			return nil, err
		}
	}
	if err := s.db.WithContext(ctx).Save(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}
