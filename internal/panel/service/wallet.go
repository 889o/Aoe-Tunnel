package service

import (
	"context"
	"errors"
	"fmt"
	"math"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"aoe-tunnel/internal/panel/model"
)

// WalletView 是钱包商城页面的账户信息。
type WalletView struct {
	Balance       float64 `json:"balance"`
	PackageID     *uint   `json:"package_id"`
	PackageName   string  `json:"package_name"`
	PackagePeriod string  `json:"package_period"`
}

type PurchasePackageInput struct {
	PackageID uint `json:"package_id" binding:"required"`
	// Period 是本次购买的计费周期，必须是套餐已配置的周期。
	Period string `json:"period" binding:"required"`
}

type PurchaseResult struct {
	Balance       float64 `json:"balance"`
	PackageID     uint    `json:"package_id"`
	PackageName   string  `json:"package_name"`
	PackagePeriod string  `json:"package_period"`
}

func (s *Service) Wallet(ctx context.Context, userID uint) (*WalletView, error) {
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	view := &WalletView{
		Balance:       user.Balance,
		PackageID:     user.PackageID,
		PackagePeriod: user.PackagePeriod,
	}
	if user.PackageID != nil {
		var pkg model.Package
		if err := s.db.WithContext(ctx).First(&pkg, *user.PackageID).Error; err == nil {
			view.PackageName = pkg.Name
		}
	}
	return view, nil
}

// PurchasePackage 在事务内扣减余额并绑定套餐与计费周期，行锁避免并发重复扣款。
func (s *Service) PurchasePackage(ctx context.Context, userID, packageID uint, period string) (*PurchaseResult, error) {
	var result PurchaseResult
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, userID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		var pkg model.Package
		if err := tx.First(&pkg, packageID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		pkg.NormalizePackagePrices()
		if !pkg.Enabled {
			return fmt.Errorf("%w: package disabled", ErrConflict)
		}
		period = normalizePeriod(period)
		price, ok := pkg.Prices[period]
		if !ok {
			return fmt.Errorf("%w: billing period not available", ErrConflict)
		}
		if user.Balance < price {
			return fmt.Errorf("%w: insufficient balance", ErrConflict)
		}
		// 金额按分对齐，避免浮点累减产生尾差。
		user.Balance = math.Round((user.Balance-price)*100) / 100
		pid := pkg.ID
		user.PackageID = &pid
		user.PackagePeriod = period
		if err := tx.Save(&user).Error; err != nil {
			return err
		}
		// 按实付金额给上级返佣，与扣款保持同一事务。
		if err := s.awardCommission(tx, &user, price, pkg.ID); err != nil {
			return err
		}
		result = PurchaseResult{
			Balance:       user.Balance,
			PackageID:     pid,
			PackageName:   pkg.Name,
			PackagePeriod: period,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func normalizePeriod(period string) string {
	if period == "" {
		return model.PackageDurationMonthly
	}
	return period
}
