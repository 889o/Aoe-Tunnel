package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"aoe-tunnel/internal/panel/model"
)

const (
	paymentChannelEpay = "epay"
	paymentChannelName = "易支付"

	epayPayTypeWechat = "wxpay"
	epayPayTypeAlipay = "alipay"

	// 充值金额按分对齐，限制单笔范围避免异常大额与 0 元订单。
	rechargeMinAmount = 0.01
	rechargeMaxAmount = 10000.0
	// defaultRechargePresets 是未配置快捷金额时钱包页展示的默认档位。
	defaultRechargePresets = "10,20,50,100,200"
)

// 易支付配置仅允许这些键；异步通知地址由面板按请求域名自动拼接，不需要手动填写。
var epayConfigKeys = map[string]string{
	"api_url":          "接口地址",
	"pid":              "商户 ID",
	"key":              "商户密钥",
	"wechat":           "微信支付开关",
	"alipay":           "支付宝开关",
	"recharge_presets": "快捷充值金额",
}

// epayHTTPClient 复用连接并统一设置超时，避免支付网关阻塞面板请求。
var epayHTTPClient = &http.Client{Timeout: 10 * time.Second}

type UpdatePaymentChannelInput struct {
	Enabled *bool             `json:"enabled,omitempty"`
	Config  map[string]string `json:"config,omitempty"`
}

// PaymentMethod 是钱包商城可用的在线支付方式。
type PaymentMethod struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

// CreatePaymentOrderInput 是发起易支付订单的入参。
type CreatePaymentOrderInput struct {
	PackageID uint   `json:"package_id" binding:"required"`
	Period    string `json:"period" binding:"required"`
	PayType   string `json:"pay_type" binding:"required"`
}

// CreateRechargeOrderInput 是余额充值下单入参，金额可自定义。
type CreateRechargeOrderInput struct {
	Amount  float64 `json:"amount" binding:"required"`
	PayType string  `json:"pay_type" binding:"required"`
}

// RechargeOptions 返回钱包页的快捷充值金额档位。
type RechargeOptions struct {
	Presets []float64 `json:"presets"`
}

// PaymentOrderView 返回跳转所需的支付地址。
type PaymentOrderView struct {
	OutTradeNo string  `json:"out_trade_no"`
	Amount     float64 `json:"amount"`
	PayType    string  `json:"pay_type"`
	Status     string  `json:"status"`
	PayURL     string  `json:"pay_url"`
	QRCode     string  `json:"qrcode"`
}

// PaymentChannels 返回支付方式列表；内置易支付在首次访问时落库，不提供删除入口。
func (s *Service) PaymentChannels(ctx context.Context) ([]model.PaymentChannel, error) {
	if err := s.ensureEpayChannel(ctx); err != nil {
		return nil, err
	}
	var channels []model.PaymentChannel
	if err := s.db.WithContext(ctx).Order("id asc").Find(&channels).Error; err != nil {
		return nil, err
	}
	for i := range channels {
		if channels[i].Config == nil {
			channels[i].Config = map[string]string{}
		}
	}
	return channels, nil
}

func (s *Service) ensureEpayChannel(ctx context.Context) error {
	var ch model.PaymentChannel
	err := s.db.WithContext(ctx).Where("code = ?", paymentChannelEpay).First(&ch).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	ch = model.PaymentChannel{
		Code:    paymentChannelEpay,
		Name:    paymentChannelName,
		Enabled: false,
		Builtin: true,
		Config:  map[string]string{},
	}
	if err := s.db.WithContext(ctx).Create(&ch).Error; err != nil {
		// 并发首次访问可能撞唯一索引，重新确认存在即可。
		var count int64
		if qerr := s.db.WithContext(ctx).Model(&model.PaymentChannel{}).
			Where("code = ?", paymentChannelEpay).Count(&count).Error; qerr != nil {
			return err
		}
		if count == 0 {
			return err
		}
	}
	return nil
}

func (s *Service) UpdatePaymentChannel(ctx context.Context, code string, in UpdatePaymentChannelInput) (*model.PaymentChannel, error) {
	code = strings.TrimSpace(code)
	if code != paymentChannelEpay {
		return nil, ErrNotFound
	}
	if err := s.ensureEpayChannel(ctx); err != nil {
		return nil, err
	}
	var ch model.PaymentChannel
	if err := s.db.WithContext(ctx).Where("code = ?", code).First(&ch).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if in.Enabled != nil {
		ch.Enabled = *in.Enabled
	}
	if in.Config != nil {
		// 合并而不是整体替换：未提交或留空的网关参数沿用已配置值，避免误删商户信息。
		merged := make(map[string]string, len(ch.Config)+len(in.Config))
		for key, value := range ch.Config {
			merged[key] = value
		}
		for rawKey, rawValue := range in.Config {
			key := strings.TrimSpace(rawKey)
			if _, ok := epayConfigKeys[key]; !ok {
				return nil, fmt.Errorf("%w: unsupported config key %q", ErrConflict, rawKey)
			}
			value := strings.TrimSpace(rawValue)
			if value == "" && (key == "api_url" || key == "pid" || key == "key") {
				continue
			}
			merged[key] = value
		}
		ch.Config = merged
	}
	if err := s.db.WithContext(ctx).Save(&ch).Error; err != nil {
		return nil, err
	}
	return &ch, nil
}

// PaymentMethods 返回已开启的易支付方式；渠道总开关关闭时返回空。
func (s *Service) PaymentMethods(ctx context.Context) ([]PaymentMethod, error) {
	ch, err := s.loadEpayChannel(ctx)
	if err != nil {
		return nil, err
	}
	if !ch.Enabled {
		return []PaymentMethod{}, nil
	}
	out := make([]PaymentMethod, 0, 2)
	if configBool(ch.Config["wechat"]) {
		out = append(out, PaymentMethod{Type: epayPayTypeWechat, Name: "微信支付"})
	}
	if configBool(ch.Config["alipay"]) {
		out = append(out, PaymentMethod{Type: epayPayTypeAlipay, Name: "支付宝"})
	}
	return out, nil
}

// CreatePaymentOrder 创建套餐订单并返回收银台地址；notify/return 地址按面板外部地址自动拼接。
func (s *Service) CreatePaymentOrder(ctx context.Context, userID uint, in CreatePaymentOrderInput, baseURL, clientIP string) (*PaymentOrderView, error) {
	ch, err := s.loadEpayChannel(ctx)
	if err != nil {
		return nil, err
	}
	payType, err := epayPayType(ch, in.PayType)
	if err != nil {
		return nil, err
	}

	var pkg model.Package
	if err := s.db.WithContext(ctx).First(&pkg, in.PackageID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	pkg.NormalizePackagePrices()
	if !pkg.Enabled {
		return nil, fmt.Errorf("%w: package disabled", ErrConflict)
	}
	period := normalizePeriod(in.Period)
	price, ok := pkg.Prices[period]
	if !ok || price <= 0 {
		return nil, fmt.Errorf("%w: billing period not available", ErrConflict)
	}

	order := model.PaymentOrder{
		OutTradeNo: newOutTradeNo(),
		UserID:     userID,
		Kind:       model.PaymentOrderKindPackage,
		PackageID:  pkg.ID,
		Period:     period,
		Amount:     roundMoney(price),
		PayType:    payType,
		Status:     model.PaymentOrderPending,
	}
	if err := s.db.WithContext(ctx).Create(&order).Error; err != nil {
		return nil, err
	}
	return s.submitEpayCheckout(ctx, ch, &order, pkg.Name+" "+periodLabel(period), baseURL, clientIP)
}

// CreateRechargeOrder 创建余额充值订单，金额可自定义；回调成功后直接增加余额。
func (s *Service) CreateRechargeOrder(ctx context.Context, userID uint, in CreateRechargeOrderInput, baseURL, clientIP string) (*PaymentOrderView, error) {
	ch, err := s.loadEpayChannel(ctx)
	if err != nil {
		return nil, err
	}
	payType, err := epayPayType(ch, in.PayType)
	if err != nil {
		return nil, err
	}
	amount := roundMoney(in.Amount)
	if amount < rechargeMinAmount || amount > rechargeMaxAmount {
		return nil, fmt.Errorf("%w: recharge amount must be between %.2f and %.2f", ErrConflict, rechargeMinAmount, rechargeMaxAmount)
	}

	order := model.PaymentOrder{
		OutTradeNo: newOutTradeNo(),
		UserID:     userID,
		Kind:       model.PaymentOrderKindRecharge,
		Amount:     amount,
		PayType:    payType,
		Status:     model.PaymentOrderPending,
	}
	if err := s.db.WithContext(ctx).Create(&order).Error; err != nil {
		return nil, err
	}
	return s.submitEpayCheckout(ctx, ch, &order, "账户余额充值", baseURL, clientIP)
}

// RechargePresets 返回快捷充值金额，未配置时使用默认档位。
func (s *Service) RechargePresets(ctx context.Context) ([]float64, error) {
	ch, err := s.loadEpayChannel(ctx)
	if err != nil {
		return nil, err
	}
	raw := strings.TrimSpace(ch.Config["recharge_presets"])
	if raw == "" {
		raw = defaultRechargePresets
	}
	return parseRechargePresets(raw), nil
}

// epayPayType 校验渠道与支付方式开关，并统一返回小写支付类型。
func epayPayType(ch *model.PaymentChannel, raw string) (string, error) {
	if !ch.Enabled {
		return "", fmt.Errorf("%w: epay channel disabled", ErrConflict)
	}
	payType := strings.ToLower(strings.TrimSpace(raw))
	switch payType {
	case epayPayTypeWechat:
		if !configBool(ch.Config["wechat"]) {
			return "", fmt.Errorf("%w: wechat pay disabled", ErrConflict)
		}
	case epayPayTypeAlipay:
		if !configBool(ch.Config["alipay"]) {
			return "", fmt.Errorf("%w: alipay disabled", ErrConflict)
		}
	default:
		return "", fmt.Errorf("%w: unsupported pay type %q", ErrConflict, payType)
	}
	if strings.TrimSpace(ch.Config["api_url"]) == "" || ch.Config["pid"] == "" || ch.Config["key"] == "" {
		return "", fmt.Errorf("%w: epay channel is not configured", ErrConflict)
	}
	return payType, nil
}

// submitEpayCheckout 向易支付下单，失败时把订单标记为 failed 以便用户重新发起。
func (s *Service) submitEpayCheckout(ctx context.Context, ch *model.PaymentChannel, order *model.PaymentOrder, subject, baseURL, clientIP string) (*PaymentOrderView, error) {
	apiURL := strings.TrimRight(strings.TrimSpace(ch.Config["api_url"]), "/")
	baseURL = strings.TrimRight(baseURL, "/")
	params := map[string]string{
		"pid":          ch.Config["pid"],
		"type":         order.PayType,
		"out_trade_no": order.OutTradeNo,
		"notify_url":   baseURL + "/api/v1/payment/epay/notify",
		"return_url":   baseURL + "/api/v1/payment/epay/return",
		"name":         subject,
		"money":        strconv.FormatFloat(order.Amount, 'f', 2, 64),
		"clientip":     clientIP,
		"sitename":     "Aoe Tunnel",
		"sign_type":    "MD5",
	}
	params["sign"] = epaySign(params, ch.Config["key"])

	res, err := epayPost(ctx, apiURL+"/mapi.php", params)
	if err != nil {
		s.markPaymentOrderFailed(ctx, order.ID)
		return nil, fmt.Errorf("%w: epay request failed: %v", ErrConflict, err)
	}
	if res.Code != 1 {
		s.markPaymentOrderFailed(ctx, order.ID)
		msg := strings.TrimSpace(res.Msg)
		if msg == "" {
			msg = "epay rejected the order"
		}
		return nil, fmt.Errorf("%w: %s", ErrConflict, msg)
	}
	return &PaymentOrderView{
		OutTradeNo: order.OutTradeNo,
		Amount:     order.Amount,
		PayType:    order.PayType,
		Status:     order.Status,
		PayURL:     res.PayURL,
		QRCode:     res.QRCode,
	}, nil
}

func newOutTradeNo() string {
	return "AO" + strings.ReplaceAll(uuid.NewString(), "-", "")[:24]
}

func roundMoney(v float64) float64 {
	return math.Round(v*100) / 100
}

// parseRechargePresets 支持中英文逗号分隔，过滤非正数、超额与重复档位。
func parseRechargePresets(raw string) []float64 {
	fields := strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == '，' })
	out := make([]float64, 0, len(fields))
	seen := make(map[int64]struct{}, len(fields))
	for _, f := range fields {
		v, err := strconv.ParseFloat(strings.TrimSpace(f), 64)
		if err != nil || v < rechargeMinAmount || v > rechargeMaxAmount {
			continue
		}
		cents := int64(math.Round(v * 100))
		if _, ok := seen[cents]; ok {
			continue
		}
		seen[cents] = struct{}{}
		out = append(out, float64(cents)/100)
	}
	return out
}

// PaymentOrder 返回用户自己的订单状态，供前端轮询支付结果。
func (s *Service) PaymentOrder(ctx context.Context, userID uint, outTradeNo string) (*model.PaymentOrder, error) {
	var order model.PaymentOrder
	if err := s.db.WithContext(ctx).
		Where("out_trade_no = ? AND user_id = ?", strings.TrimSpace(outTradeNo), userID).
		First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &order, nil
}

// HandleEpayNotify 校验收款通知签名并在事务内绑定套餐；重复通知按幂等成功处理。
func (s *Service) HandleEpayNotify(ctx context.Context, params map[string]string) error {
	ch, err := s.loadEpayChannel(ctx)
	if err != nil {
		return err
	}
	pid := strings.TrimSpace(ch.Config["pid"])
	if pid == "" || params["pid"] != pid {
		return fmt.Errorf("%w: pid mismatch", ErrUnauthorized)
	}
	sign := params["sign"]
	if sign == "" || !strings.EqualFold(sign, epaySign(params, ch.Config["key"])) {
		return fmt.Errorf("%w: signature mismatch", ErrUnauthorized)
	}
	if params["trade_status"] != "TRADE_SUCCESS" {
		return nil
	}
	outTradeNo := strings.TrimSpace(params["out_trade_no"])
	if outTradeNo == "" {
		return fmt.Errorf("%w: out_trade_no required", ErrConflict)
	}
	amount, err := strconv.ParseFloat(strings.TrimSpace(params["money"]), 64)
	if err != nil {
		return fmt.Errorf("%w: invalid money", ErrConflict)
	}
	tradeNo := strings.TrimSpace(params["trade_no"])

	return s.applyPaidOrder(ctx, outTradeNo, amount, tradeNo)
}

// applyPaidOrder 在事务内幂等地把订单置为已支付：充值加余额，套餐订单绑定周期。
func (s *Service) applyPaidOrder(ctx context.Context, outTradeNo string, amount float64, tradeNo string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order model.PaymentOrder
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("out_trade_no = ?", outTradeNo).First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		if order.Status == model.PaymentOrderPaid {
			return nil
		}
		if int64(math.Round(amount*100)) != int64(math.Round(order.Amount*100)) {
			return fmt.Errorf("%w: amount mismatch", ErrConflict)
		}
		var user model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, order.UserID).Error; err != nil {
			return err
		}
		if order.Kind == model.PaymentOrderKindRecharge {
			// 充值只增加余额，不改变用户当前套餐绑定。
			user.Balance = roundMoney(user.Balance + order.Amount)
		} else {
			pid := order.PackageID
			user.PackageID = &pid
			user.PackagePeriod = order.Period
			// 套餐支付成功才返佣；重复回调由上面的已支付判断挡住。
			if err := s.awardCommission(tx, &user, order.Amount, pid); err != nil {
				return err
			}
		}
		if err := tx.Save(&user).Error; err != nil {
			return err
		}
		now := time.Now()
		order.Status = model.PaymentOrderPaid
		order.TradeNo = tradeNo
		order.PaidAt = &now
		return tx.Save(&order).Error
	})
}

// SyncPaymentOrder 主动向网关查询待支付订单；异步通知不可达时仍能确认到账。
func (s *Service) SyncPaymentOrder(ctx context.Context, userID uint, outTradeNo string) (*model.PaymentOrder, error) {
	order, err := s.PaymentOrder(ctx, userID, outTradeNo)
	if err != nil || order.Status != model.PaymentOrderPending {
		return order, err
	}
	s.syncPendingOrder(ctx, order)
	return s.PaymentOrder(ctx, userID, outTradeNo)
}

// SyncPaymentOrderByNo 供同步跳转回调查询订单，参数来自网关查询结果而非用户输入。
func (s *Service) SyncPaymentOrderByNo(ctx context.Context, outTradeNo string) {
	outTradeNo = strings.TrimSpace(outTradeNo)
	if outTradeNo == "" {
		return
	}
	var order model.PaymentOrder
	if err := s.db.WithContext(ctx).Where("out_trade_no = ?", outTradeNo).First(&order).Error; err != nil {
		return
	}
	s.syncPendingOrder(ctx, &order)
}

func (s *Service) syncPendingOrder(ctx context.Context, order *model.PaymentOrder) {
	if order.Status != model.PaymentOrderPending {
		return
	}
	ch, err := s.loadEpayChannel(ctx)
	if err != nil || !epayQueryReady(ch) || !epaySyncAllowed(order.OutTradeNo) {
		return
	}
	paid, amount, tradeNo, err := queryEpayOrder(ctx, ch, order.OutTradeNo)
	if err != nil {
		s.log.Warn("query epay order", zap.String("out_trade_no", order.OutTradeNo), zap.Error(err))
		return
	}
	if !paid {
		return
	}
	if err := s.applyPaidOrder(ctx, order.OutTradeNo, amount, tradeNo); err != nil {
		s.log.Warn("apply synced epay order", zap.String("out_trade_no", order.OutTradeNo), zap.Error(err))
	}
}

// queryEpayOrder 调用易支付订单查询接口，返回是否已支付、金额与网关流水号。
func queryEpayOrder(ctx context.Context, ch *model.PaymentChannel, outTradeNo string) (bool, float64, string, error) {
	base := strings.TrimRight(strings.TrimSpace(ch.Config["api_url"]), "/")
	q := url.Values{}
	q.Set("act", "order")
	q.Set("pid", ch.Config["pid"])
	q.Set("key", ch.Config["key"])
	q.Set("out_trade_no", outTradeNo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api.php?"+q.Encode(), nil)
	if err != nil {
		return false, 0, "", err
	}
	resp, err := epayHTTPClient.Do(req)
	if err != nil {
		return false, 0, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, 0, "", fmt.Errorf("gateway status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return false, 0, "", err
	}
	var res struct {
		Code       int             `json:"code"`
		TradeNo    string          `json:"trade_no"`
		OutTradeNo string          `json:"out_trade_no"`
		Money      json.RawMessage `json:"money"`
		Status     json.RawMessage `json:"status"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return false, 0, "", fmt.Errorf("decode gateway response: %w", err)
	}
	if res.Code != 1 {
		return false, 0, "", nil
	}
	status := strings.Trim(strings.TrimSpace(string(res.Status)), `"`)
	if status != "1" {
		return false, 0, "", nil
	}
	if echo := strings.TrimSpace(res.OutTradeNo); echo != "" && echo != outTradeNo {
		return false, 0, "", fmt.Errorf("out_trade_no mismatch")
	}
	moneyRaw := strings.Trim(strings.TrimSpace(string(res.Money)), `"`)
	amount, err := strconv.ParseFloat(moneyRaw, 64)
	if err != nil {
		return false, 0, "", fmt.Errorf("invalid money %q", moneyRaw)
	}
	return true, amount, strings.TrimSpace(res.TradeNo), nil
}

func epayQueryReady(ch *model.PaymentChannel) bool {
	return strings.TrimSpace(ch.Config["api_url"]) != "" && ch.Config["pid"] != "" && ch.Config["key"] != ""
}

var (
	epaySyncMu sync.Mutex
	epaySyncAt = make(map[string]time.Time)
)

// epaySyncAllowed 限制同一订单的主动查询频率，避免前端轮询把网关打满。
func epaySyncAllowed(outTradeNo string) bool {
	epaySyncMu.Lock()
	defer epaySyncMu.Unlock()
	now := time.Now()
	if last, ok := epaySyncAt[outTradeNo]; ok && now.Sub(last) < 5*time.Second {
		return false
	}
	if len(epaySyncAt) > 10000 {
		epaySyncAt = make(map[string]time.Time)
	}
	epaySyncAt[outTradeNo] = now
	return true
}

func (s *Service) loadEpayChannel(ctx context.Context) (*model.PaymentChannel, error) {
	if err := s.ensureEpayChannel(ctx); err != nil {
		return nil, err
	}
	var ch model.PaymentChannel
	if err := s.db.WithContext(ctx).Where("code = ?", paymentChannelEpay).First(&ch).Error; err != nil {
		return nil, err
	}
	if ch.Config == nil {
		ch.Config = map[string]string{}
	}
	return &ch, nil
}

func (s *Service) markPaymentOrderFailed(ctx context.Context, id uint) {
	if err := s.db.WithContext(ctx).Model(&model.PaymentOrder{}).Where("id = ?", id).
		Update("status", model.PaymentOrderFailed).Error; err != nil {
		s.log.Warn("mark payment order failed", zap.Uint("order_id", id), zap.Error(err))
	}
}

func configBool(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "on", "yes":
		return true
	default:
		return false
	}
}

func periodLabel(period string) string {
	switch period {
	case model.PackageDurationQuarterly:
		return "季付"
	case model.PackageDurationHalfYear:
		return "半年付"
	case model.PackageDurationYearly:
		return "年付"
	case model.PackageDurationPermanent:
		return "永久"
	default:
		return "月付"
	}
}

// epaySign 按彩虹易支付规则：排除 sign/sign_type 与空值，参数名升序拼接后追加密钥再 MD5。
func epaySign(params map[string]string, key string) string {
	keys := make([]string, 0, len(params))
	for k, v := range params {
		if k == "sign" || k == "sign_type" || v == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(params[k])
	}
	b.WriteString(key)
	sum := md5.Sum([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

type epayResponse struct {
	Code   int    `json:"code"`
	Msg    string `json:"msg"`
	PayURL string `json:"payurl"`
	QRCode string `json:"qrcode"`
}

func epayPost(ctx context.Context, endpoint string, params map[string]string) (*epayResponse, error) {
	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := epayHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gateway status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	var out epayResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode gateway response: %w", err)
	}
	return &out, nil
}
