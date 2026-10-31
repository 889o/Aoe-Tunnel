package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"aoe-tunnel/internal/panel/auth"
	"aoe-tunnel/internal/panel/service"
)

const identityKey = "aoe.identity"

// AuthDeps 汇总鉴权所需的依赖：JWT 用于 Web 与 Agent 业务调用，节点令牌只用于上线接口。
type AuthDeps struct {
	Svc    *service.Service
	Tokens *auth.Manager
}

func Identity(c *gin.Context) *service.Principal {
	v, ok := c.Get(identityKey)
	if !ok {
		return nil
	}
	p, _ := v.(*service.Principal)
	return p
}

func setIdentity(c *gin.Context, p *service.Principal) {
	c.Set(identityKey, p)
}

// JWT 接受 Bearer access token，Web 用户与 Agent 共用同一条通道。
func (d *AuthDeps) JWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		p, err := d.fromJWT(c)
		if err != nil {
			Fail(c, http.StatusUnauthorized, "登录已失效，请重新登录")
			c.Abort()
			return
		}
		setIdentity(c, p)
		c.Next()
	}
}

// NodeToken 只用于节点上线校验：令牌必须与路径中的节点编码一致，不具备任何业务权限。
func (d *AuthDeps) NodeToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		p, err := d.Svc.AuthenticateNodeToken(c.Request.Context(), c.GetHeader("X-Node-Token"))
		if err != nil || p.NodeCode != c.Param("code") {
			Fail(c, http.StatusUnauthorized, "节点令牌无效")
			c.Abort()
			return
		}
		setIdentity(c, p)
		c.Next()
	}
}

// RequirePerm 在鉴权之后校验权限码，缺权限返回 403。
func RequirePerm(code string) gin.HandlerFunc {
	return func(c *gin.Context) {
		p := Identity(c)
		if p == nil {
			Fail(c, http.StatusUnauthorized, "未鉴权")
			c.Abort()
			return
		}
		if !p.Has(code) {
			Fail(c, http.StatusForbidden, "缺少权限 "+code)
			c.Abort()
			return
		}
		c.Next()
	}
}

func (d *AuthDeps) fromJWT(c *gin.Context) (*service.Principal, error) {
	raw := strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
	if raw == "" {
		return nil, service.ErrUnauthorized
	}
	claims, err := d.Tokens.Parse(raw, auth.TokenTypeAccess)
	if err != nil {
		return nil, service.ErrUnauthorized
	}
	// 每次请求校验会话仍有效，踢下线后旧 access token 立即失效。
	if !d.Svc.SessionValid(c.Request.Context(), claims.SID) {
		return nil, service.ErrUnauthorized
	}
	p, err := d.Svc.LoadPrincipal(c.Request.Context(), claims.UID, "jwt")
	if err != nil {
		return nil, err
	}
	p.SID = claims.SID
	return p, nil
}
