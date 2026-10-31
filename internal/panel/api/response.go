package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"aoe-tunnel/internal/panel/service"
)

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok", "data": data})
}

func Fail(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"code": status, "message": message})
}

func handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrUnauthorized):
		Fail(c, http.StatusUnauthorized, "用户名或密码错误")
	case errors.Is(err, service.ErrForbidden):
		Fail(c, http.StatusForbidden, err.Error())
	case errors.Is(err, service.ErrNotFound):
		Fail(c, http.StatusNotFound, "not found")
	case errors.Is(err, service.ErrConflict):
		Fail(c, http.StatusConflict, err.Error())
	default:
		Fail(c, http.StatusBadRequest, err.Error())
	}
}
