package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	// TokenTypeAccess 与 TokenTypeRefresh 写进 claim，避免 refresh token 被当成 access 使用。
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

var ErrInvalidToken = errors.New("auth: invalid token")

type Claims struct {
	Type string `json:"typ"`
	UID  uint   `json:"uid"`
	SID  string `json:"sid"`
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// Manager 负责签发/校验 JWT，access 与 refresh 用同一密钥但强制区分 typ。
type Manager struct {
	secret     []byte
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

func NewManager(secret []byte, accessTTL, refreshTTL time.Duration) *Manager {
	return &Manager{secret: secret, AccessTTL: accessTTL, RefreshTTL: refreshTTL}
}

func (m *Manager) SignAccess(uid uint, sid, role string) (string, error) {
	return m.sign(uid, sid, role, TokenTypeAccess, m.AccessTTL)
}

func (m *Manager) SignRefresh(uid uint, sid string) (string, error) {
	return m.sign(uid, sid, "", TokenTypeRefresh, m.RefreshTTL)
}

func (m *Manager) sign(uid uint, sid, role, typ string, ttl time.Duration) (string, error) {
	now := time.Now()
	// jti 必须随机：同一秒内重复签发否则会得到完全相同的 refresh token，导致轮换后旧 token 仍可复用。
	jti, err := randomHex(12)
	if err != nil {
		return "", err
	}
	claims := Claims{
		Type: typ,
		UID:  uid,
		SID:  sid,
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-10 * time.Second)),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			Issuer:    "aoe-panel",
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

func (m *Manager) Parse(token, wantType string) (*Claims, error) {
	claims := &Claims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
		}
		return m.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer("aoe-panel"))
	if err != nil || !parsed.Valid {
		return nil, ErrInvalidToken
	}
	if claims.Type != wantType {
		return nil, ErrInvalidToken
	}
	if claims.SID == "" || claims.UID == 0 {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// DeriveSecret 在没有显式配置密钥时，从稳定的部署信息派生，保证重启后已签发 token 不失效。
func DeriveSecret(label string) []byte {
	sum := sha256.Sum256([]byte("aoe-panel:" + label))
	return sum[:]
}

func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func VerifyPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

// HashToken 用于 refresh token：数据库只存 sha256，泄露库也无法直接使用。
func HashToken(tok string) string {
	sum := sha256.Sum256([]byte(tok))
	return hex.EncodeToString(sum[:])
}

func NewSID() (string, error) {
	b, err := randomHex(16)
	if err != nil {
		return "", err
	}
	return b, nil
}

func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
