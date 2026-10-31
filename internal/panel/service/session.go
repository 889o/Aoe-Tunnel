package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"gorm.io/gorm"

	"aoe-tunnel/internal/panel/model"
)

// sessionCacheTTL 只用于衰减「会话是否仍有效」的查询次数；本进程内撤销会主动清缓存，踢下线即时生效。
const sessionCacheTTL = 5 * time.Second

// revokedTTL 是 JWT 黑名单的保留时长，只需覆盖 access token 的最长有效期。
const revokedTTL = time.Hour

type sessionCacheEntry struct {
	valid   bool
	checked time.Time
}

type sessionGuard struct {
	mu      sync.Mutex
	m       map[string]sessionCacheEntry
	revoked map[string]time.Time
}

func newSessionGuard() *sessionGuard {
	return &sessionGuard{
		m:       make(map[string]sessionCacheEntry),
		revoked: make(map[string]time.Time),
	}
}

func (g *sessionGuard) get(sid string) (bool, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	e, ok := g.m[sid]
	if !ok || time.Since(e.checked) > sessionCacheTTL {
		return false, false
	}
	return e.valid, true
}

func (g *sessionGuard) set(sid string, valid bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.m[sid] = sessionCacheEntry{valid: valid, checked: time.Now()}
}

func (g *sessionGuard) drop(sid string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.m, sid)
}

// revoke 把 SID 写入 JWT 黑名单，即使缓存被清掉也不会回落到数据库的旧状态。
func (g *sessionGuard) revoke(sid string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.m, sid)
	g.revoked[sid] = time.Now().Add(revokedTTL)
	g.cleanupLocked()
}

func (g *sessionGuard) isRevoked(sid string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	exp, ok := g.revoked[sid]
	if !ok {
		return false
	}
	if time.Now().After(exp) {
		delete(g.revoked, sid)
		return false
	}
	return true
}

func (g *sessionGuard) cleanupLocked() {
	now := time.Now()
	for sid, exp := range g.revoked {
		if now.After(exp) {
			delete(g.revoked, sid)
		}
	}
}

// SessionValid 校验会话未被撤销且未过期，供 JWT 中间件每次请求判定。
func (s *Service) SessionValid(ctx context.Context, sid string) bool {
	// 踢下线立即失效：黑名单优先于任何缓存与数据库查询。
	if s.sessions.isRevoked(sid) {
		return false
	}
	if valid, ok := s.sessions.get(sid); ok {
		return valid
	}
	var sess model.Session
	err := s.db.WithContext(ctx).Where("sid = ?", sid).First(&sess).Error
	if err != nil {
		// 会话不存在与查询失败一律按无效处理，避免 fail-open。
		s.sessions.set(sid, false)
		return false
	}
	valid := sess.RevokedAt == nil && time.Now().Before(sess.ExpiresAt)
	s.sessions.set(sid, valid)
	return valid
}

func (s *Service) createSession(ctx context.Context, userID uint, sid, refreshHash string, ttl time.Duration, ua, ip string) error {
	now := time.Now()
	sess := model.Session{
		SID:         sid,
		UserID:      userID,
		RefreshHash: refreshHash,
		UA:          ua,
		IP:          ip,
		ExpiresAt:   now.Add(ttl),
		LastSeenAt:  &now,
	}
	return s.db.WithContext(ctx).Create(&sess).Error
}

// rotateSession 校验 refresh 哈希后轮换为新的 refresh token，旧 token 立即失效。
func (s *Service) rotateSession(ctx context.Context, sid, oldHash, newHash string) (bool, error) {
	res := s.db.WithContext(ctx).Model(&model.Session{}).
		Where("sid = ? AND refresh_hash = ? AND revoked_at IS NULL AND expires_at > ?", sid, oldHash, time.Now()).
		Updates(map[string]any{"refresh_hash": newHash, "last_seen_at": time.Now()})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (s *Service) revokeSession(ctx context.Context, sid string) error {
	res := s.db.WithContext(ctx).Model(&model.Session{}).
		Where("sid = ? AND revoked_at IS NULL", sid).
		Update("revoked_at", time.Now())
	if res.Error != nil {
		return res.Error
	}
	s.sessions.revoke(sid)
	return nil
}

func (s *Service) revokeAllSessions(ctx context.Context, userID uint) (int64, error) {
	var sids []string
	if err := s.db.WithContext(ctx).Model(&model.Session{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).Pluck("sid", &sids).Error; err != nil {
		return 0, err
	}
	res := s.db.WithContext(ctx).Model(&model.Session{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", time.Now())
	if res.Error != nil {
		return 0, res.Error
	}
	for _, sid := range sids {
		s.sessions.revoke(sid)
	}
	return res.RowsAffected, nil
}

type SessionView struct {
	model.Session
	Username string `json:"username"`
	Current  bool   `json:"current"`
}

// ListSessions 当 userID 非空时只返回该用户的会话，供无 session:manage-all 权限的角色使用。
func (s *Service) ListSessions(ctx context.Context, userID *uint, currentSID string) ([]SessionView, error) {
	q := s.db.WithContext(ctx).Order("id desc")
	if userID != nil {
		q = q.Where("user_id = ?", *userID)
	}
	q = q.Where("revoked_at IS NULL")
	var sessions []model.Session
	if err := q.Find(&sessions).Error; err != nil {
		return nil, err
	}

	usernames := make(map[uint]string, len(sessions))
	out := make([]SessionView, 0, len(sessions))
	for _, sess := range sessions {
		name, ok := usernames[sess.UserID]
		if !ok {
			var u model.User
			if err := s.db.WithContext(ctx).Select("username").First(&u, sess.UserID).Error; err == nil {
				name = u.Username
			}
			usernames[sess.UserID] = name
		}
		out = append(out, SessionView{Session: sess, Username: name, Current: sess.SID == currentSID})
	}
	return out, nil
}

func (s *Service) getSession(ctx context.Context, sid string) (*model.Session, error) {
	var sess model.Session
	err := s.db.WithContext(ctx).Where("sid = ?", sid).First(&sess).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

// KickSession 撤销指定会话；无 session:manage-all 权限时只能踢自己的会话。
func (s *Service) KickSession(ctx context.Context, sid string, actor *Principal) error {
	sess, err := s.getSession(ctx, sid)
	if err != nil {
		return err
	}
	if sess.UserID != actor.User.ID && !actor.Has(PermSessionManageAll) {
		return ErrForbidden
	}
	return s.revokeSession(ctx, sid)
}
