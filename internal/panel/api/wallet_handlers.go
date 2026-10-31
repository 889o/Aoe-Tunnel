package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"aoe-tunnel/internal/panel/service"
)

func (h *Handler) wallet(c *gin.Context) {
	p := Identity(c)
	if p == nil {
		Fail(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	view, err := h.svc.Wallet(c.Request.Context(), p.User.ID)
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, view)
}

func (h *Handler) purchasePackage(c *gin.Context) {
	p := Identity(c)
	if p == nil {
		Fail(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	var in service.PurchasePackageInput
	if err := c.ShouldBindJSON(&in); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	res, err := h.svc.PurchasePackage(c.Request.Context(), p.User.ID, in.PackageID, in.Period)
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, res)
}

func (h *Handler) paymentMethods(c *gin.Context) {
	methods, err := h.svc.PaymentMethods(c.Request.Context())
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, methods)
}

func (h *Handler) createPaymentOrder(c *gin.Context) {
	p := Identity(c)
	if p == nil {
		Fail(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	var in service.CreatePaymentOrderInput
	if err := c.ShouldBindJSON(&in); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	view, err := h.svc.CreatePaymentOrder(c.Request.Context(), p.User.ID, in, h.publicBaseURL(c), c.ClientIP())
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, view)
}

func (h *Handler) rechargeOptions(c *gin.Context) {
	presets, err := h.svc.RechargePresets(c.Request.Context())
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, service.RechargeOptions{Presets: presets})
}

func (h *Handler) createRechargeOrder(c *gin.Context) {
	p := Identity(c)
	if p == nil {
		Fail(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	var in service.CreateRechargeOrderInput
	if err := c.ShouldBindJSON(&in); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	view, err := h.svc.CreateRechargeOrder(c.Request.Context(), p.User.ID, in, h.publicBaseURL(c), c.ClientIP())
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, view)
}

func (h *Handler) paymentOrder(c *gin.Context) {
	p := Identity(c)
	if p == nil {
		Fail(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	order, err := h.svc.SyncPaymentOrder(c.Request.Context(), p.User.ID, c.Param("no"))
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, order)
}

// epayNotify 处理易支付异步通知，返回纯文本 success/fail 以符合网关约定。
func (h *Handler) epayNotify(c *gin.Context) {
	if err := c.Request.ParseForm(); err != nil {
		c.String(http.StatusOK, "fail")
		return
	}
	params := make(map[string]string, len(c.Request.Form))
	for key, values := range c.Request.Form {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}
	if err := h.svc.HandleEpayNotify(c.Request.Context(), params); err != nil {
		c.String(http.StatusOK, "fail")
		return
	}
	c.String(http.StatusOK, "success")
}

// epayReturn 是同步跳转地址，统一回到钱包商城的支付结果页。
func (h *Handler) epayReturn(c *gin.Context) {
	// 异步通知可能被内网/防火墙阻断，同步跳转回来时按订单号主动核对一次支付结果。
	if err := c.Request.ParseForm(); err == nil {
		h.svc.SyncPaymentOrderByNo(c.Request.Context(), c.Request.Form.Get("out_trade_no"))
	}
	c.Redirect(http.StatusFound, h.publicBaseURL(c)+"/#/store?paid=1")
}


