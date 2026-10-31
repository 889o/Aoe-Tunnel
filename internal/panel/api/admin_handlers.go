package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"aoe-tunnel/internal/panel/service"
)

func (h *Handler) listUsers(c *gin.Context) {
	users, err := h.svc.ListUsers(c.Request.Context())
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, users)
}

func (h *Handler) createUser(c *gin.Context) {
	var in service.CreateUserInput
	if err := c.ShouldBindJSON(&in); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	user, err := h.svc.CreateUser(c.Request.Context(), in)
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, user)
}

func (h *Handler) updateUser(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	var in service.UpdateUserInput
	if err := c.ShouldBindJSON(&in); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	user, err := h.svc.UpdateUser(c.Request.Context(), id, in, Identity(c))
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, user)
}

func (h *Handler) deleteUser(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.DeleteUser(c.Request.Context(), id, Identity(c)); err != nil {
		handleErr(c, err)
		return
	}
	OK(c, gin.H{"deleted": id})
}

func (h *Handler) listRoles(c *gin.Context) {
	roles, err := h.svc.ListRoles(c.Request.Context())
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, roles)
}

func (h *Handler) createRole(c *gin.Context) {
	var in service.CreateRoleInput
	if err := c.ShouldBindJSON(&in); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	role, err := h.svc.CreateRole(c.Request.Context(), in)
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, role)
}

func (h *Handler) updateRole(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	var in service.UpdateRoleInput
	if err := c.ShouldBindJSON(&in); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	role, err := h.svc.UpdateRole(c.Request.Context(), id, in)
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, role)
}

func (h *Handler) deleteRole(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.DeleteRole(c.Request.Context(), id); err != nil {
		handleErr(c, err)
		return
	}
	OK(c, gin.H{"deleted": id})
}

func (h *Handler) listPermissions(c *gin.Context) {
	perms, err := h.svc.ListPermissions(c.Request.Context())
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, perms)
}

func (h *Handler) listSessions(c *gin.Context) {
	p := Identity(c)
	var userID *uint
	if !p.Has(service.PermSessionManageAll) {
		userID = &p.User.ID
	}
	sessions, err := h.svc.ListSessions(c.Request.Context(), userID, p.SID)
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, sessions)
}

func (h *Handler) kickSession(c *gin.Context) {
	sid := c.Param("sid")
	if sid == "" {
		Fail(c, http.StatusBadRequest, "invalid session id")
		return
	}
	if err := h.svc.KickSession(c.Request.Context(), sid, Identity(c)); err != nil {
		handleErr(c, err)
		return
	}
	OK(c, gin.H{"kicked": sid})
}

type profileInput struct {
	DisplayName *string `json:"display_name,omitempty" binding:"omitempty,max=64"`
	OldPassword *string `json:"old_password,omitempty" binding:"omitempty,min=1,max=72"`
	NewPassword *string `json:"new_password,omitempty" binding:"omitempty,min=6,max=72"`
}

func (h *Handler) getProfile(c *gin.Context) {
	p := Identity(c)
	view, role, err := h.svc.Profile(c.Request.Context(), p.User.ID)
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, gin.H{"user": view, "role": role, "permissions": p.Permissions})
}

func (h *Handler) inviteInfo(c *gin.Context) {
	p := Identity(c)
	info, err := h.svc.InviteInfo(c.Request.Context(), p.User.ID, h.publicBaseURL(c))
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, info)
}

func (h *Handler) updateProfile(c *gin.Context) {
	var in profileInput
	if err := c.ShouldBindJSON(&in); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	user, err := h.svc.UpdateProfile(c.Request.Context(), Identity(c).User.ID, in.DisplayName, in.OldPassword, in.NewPassword)
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, user)
}

func (h *Handler) systemSettings(c *gin.Context) {
	cfg, err := h.svc.SiteConfig(c.Request.Context())
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, cfg)
}

func (h *Handler) updateSystemSettings(c *gin.Context) {
	var in service.UpdateSystemSettingsInput
	if err := c.ShouldBindJSON(&in); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	cfg, err := h.svc.UpdateSystemSettings(c.Request.Context(), in)
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, cfg)
}

func (h *Handler) paymentChannels(c *gin.Context) {
	channels, err := h.svc.PaymentChannels(c.Request.Context())
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, channels)
}

func (h *Handler) updatePaymentChannel(c *gin.Context) {
	var in service.UpdatePaymentChannelInput
	if err := c.ShouldBindJSON(&in); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	channel, err := h.svc.UpdatePaymentChannel(c.Request.Context(), c.Param("code"), in)
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, channel)
}

func (h *Handler) dashboard(c *gin.Context) {
	summary, err := h.svc.DashboardSummary(c.Request.Context(), Identity(c))
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, summary)
}
