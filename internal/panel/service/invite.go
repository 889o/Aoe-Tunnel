package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"aoe-tunnel/internal/panel/model"
)

// InviteInfo 是个人设置页「邀请链接」模块展示的数据。
type InviteInfo struct {
	Enabled         bool    `json:"enabled"`
	Ratio           float64 `json:"ratio"`
	Code            string  `json:"code"`
	URL             string  `json:"url"`
	InvitedCount    int64   `json:"invited_count"`
	TotalCommission float64 `json:"total_commission"`
}

const inviteCodeAlphabet = "abcdefghjkmnpqrstuvwxyz23456789"

func randomInviteCode() (string, error) {
	buf := make([]byte, 8)
	for i := range buf {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(inviteCodeAlphabet))))
		if err != nil {
			return "", err
		}
		buf[i] = inviteCodeAlphabet[n.Int64()]
	}
	return string(buf), nil
}

// ensureInviteCode 为存量用户补发邀请码；空码重试直到不冲突。
func (s *Service) ensureInviteCode(ctx context.Context, user *model.User) (string, error) {
	if user.InviteCode != "" {
		return user.InviteCode, nil
	}
	for attempt := 0; attempt < 5; attempt++ {
		code, err := randomInviteCode()
		if err != nil {
			return "", err
		}
		var count int64
		if err := s.db.WithContext(ctx).Model(&model.User{}).Where("invite_code = ?", code).Count(&count).Error; err != nil {
			return "", err
		}
		if count > 0 {
			continue
		}
		if err := s.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", user.ID).
			Update("invite_code", code).Error; err != nil {
			return "", err
		}
		user.InviteCode = code
		return code, nil
	}
	return "", fmt.Errorf("%w: failed to allocate invite code", ErrConflict)
}

// InviteInfo 返回当前用户的邀请码、邀请链接与累计收益。
func (s *Service) InviteInfo(ctx context.Context, userID uint, baseURL string) (*InviteInfo, error) {
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	code, err := s.ensureInviteCode(ctx, &user)
	if err != nil {
		return nil, err
	}
	cfg, err := s.SiteConfig(ctx)
	if err != nil {
		return nil, err
	}
	var invited int64
	if err := s.db.WithContext(ctx).Model(&model.User{}).Where("inviter_id = ?", userID).Count(&invited).Error; err != nil {
		return nil, err
	}
	var total float64
	if err := s.db.WithContext(ctx).Model(&model.CommissionRecord{}).
		Where("inviter_id = ?", userID).
		Select("COALESCE(SUM(commission), 0)").Scan(&total).Error; err != nil {
		return nil, err
	}
	base := strings.TrimSuffix(strings.TrimSpace(baseURL), "/")
	link := ""
	if base != "" {
		link = base + "/#/register?invite=" + code
	}
	return &InviteInfo{
		Enabled:         cfg.DistributionOpen,
		Ratio:           cfg.DistributionRatio,
		Code:            code,
		URL:             link,
		InvitedCount:    invited,
		TotalCommission: math.Round(total*100) / 100,
	}, nil
}

// awardCommission 在购买套餐的同一事务内给上级返佣；分销关闭、无上级或上级非活跃时跳过。
func (s *Service) awardCommission(tx *gorm.DB, buyer *model.User, amount float64, packageID uint) error {
	if buyer.InviterID == nil || amount <= 0 {
		return nil
	}
	var cfg model.SiteConfig
	if err := tx.First(&cfg, 1).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if !cfg.DistributionOpen || cfg.DistributionRatio <= 0 {
		return nil
	}
	var inviter model.User
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&inviter, *buyer.InviterID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if inviter.Status != "active" || inviter.ID == buyer.ID {
		return nil
	}
	commission := math.Round(amount*cfg.DistributionRatio) / 100
	if commission <= 0 {
		return nil
	}
	inviter.Balance = roundMoney(inviter.Balance + commission)
	if err := tx.Model(&model.User{}).Where("id = ?", inviter.ID).Update("balance", inviter.Balance).Error; err != nil {
		return err
	}
	return tx.Create(&model.CommissionRecord{
		InviterID:  inviter.ID,
		BuyerID:    buyer.ID,
		PackageID:  packageID,
		Amount:     amount,
		Ratio:      cfg.DistributionRatio,
		Commission: commission,
	}).Error
}
