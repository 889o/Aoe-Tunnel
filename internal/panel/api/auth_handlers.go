package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"aoe-tunnel/internal/panel/model"
	"aoe-tunnel/internal/panel/service"
)

type loginInput struct {
	Username string `json:"username" binding:"required,max=64"`
	Password string `json:"password" binding:"required,min=1,max=72"`
}

type loginOutput struct {
	AccessToken  string             `json:"access_token"`
	RefreshToken string             `json:"refresh_token"`
	TokenType    string             `json:"token_type"`
	ExpiresIn    int64              `json:"expires_in"`
	User         model.User         `json:"user"`
	Role         service.RoleBrief  `json:"role"`
	Permissions  []string           `json:"permissions"`
	Menus        []service.MenuItem `json:"menus"`
}

func (h *Handler) login(c *gin.Context) {
	var in loginInput
	if err := c.ShouldBindJSON(&in); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	tokens, p, err := h.svc.Login(c.Request.Context(), in.Username, in.Password,
		truncate(c.GetHeader("User-Agent"), 255), c.ClientIP())
	if err != nil {
		handleErr(c, err)
		return
	}
	me := h.svc.Me(p)
	OK(c, loginOutput{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    tokens.TokenType,
		ExpiresIn:    tokens.ExpiresIn,
		User:         me.User,
		Role:         me.Role,
		Permissions:  me.Permissions,
		Menus:        me.Menus,
	})
}

type refreshInput struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type registerInput struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Password string `json:"password" binding:"required,min=6,max=72"`
	// InviteCode 来自邀请链接，仅分销开启时用于绑定上级。
	InviteCode string `json:"invite_code,omitempty" binding:"max=16"`
}

func (h *Handler) register(c *gin.Context) {
	var in registerInput
	if err := c.ShouldBindJSON(&in); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	user, err := h.svc.Register(c.Request.Context(), in.Username, in.Password, in.InviteCode)
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, gin.H{"username": user.Username, "status": user.Status})
}

// publicSiteConfig 无需登录，供登录/注册页展示站点标题与注册开关。
func (h *Handler) publicSiteConfig(c *gin.Context) {
	cfg, err := h.svc.PublicSiteConfig(c.Request.Context())
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, cfg)
}

func (h *Handler) refresh(c *gin.Context) {
	var in refreshInput
	if err := c.ShouldBindJSON(&in); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	tokens, _, err := h.svc.Refresh(c.Request.Context(), in.RefreshToken)
	if err != nil {
		Fail(c, http.StatusUnauthorized, "登录已失效，请重新登录")
		return
	}
	OK(c, tokens)
}

func (h *Handler) logout(c *gin.Context) {
	p := Identity(c)
	if p == nil {
		Fail(c, http.StatusUnauthorized, "未鉴权")
		return
	}
	if err := h.svc.Logout(c.Request.Context(), p.SID); err != nil {
		handleErr(c, err)
		return
	}
	OK(c, gin.H{"logged_out": true})
}

func (h *Handler) me(c *gin.Context) {
	p := Identity(c)
	if p == nil {
		Fail(c, http.StatusUnauthorized, "未鉴权")
		return
	}
	OK(c, h.svc.Me(p))
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n]
}
