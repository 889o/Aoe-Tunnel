package service

import (
	"context"
	"time"

	"aoe-tunnel/internal/panel/model"
)

type TrafficTotals struct {
	UpBytes   int64 `json:"up_bytes"`
	DownBytes int64 `json:"down_bytes"`
	BillBytes int64 `json:"bill_bytes"`
}

type TrafficBucket struct {
	Bucket    string `json:"bucket"`
	UpBytes   int64  `json:"up_bytes"`
	DownBytes int64  `json:"down_bytes"`
	BillBytes int64  `json:"bill_bytes"`
}

// DashboardPackage 是当前登录用户套餐的用量概要与剩余额度。
type DashboardPackage struct {
	Name              string  `json:"name"`
	Period            string  `json:"period"`
	RuleLimit         int     `json:"rule_limit"`
	RuleUsed          int     `json:"rule_used"`
	RuleRemaining     int     `json:"rule_remaining"`
	TrafficLimitByte  int64   `json:"traffic_limit_bytes"`
	TrafficUsedByte   int64   `json:"traffic_used_bytes"`
	TrafficRemainByte int64   `json:"traffic_remaining_bytes"`
	TrafficLimitGB    float64 `json:"traffic_limit_gb"`
}

type DashboardSummary struct {
	Package    DashboardPackage `json:"package"`
	Traffic24h TrafficTotals    `json:"traffic_24h"`
	Series     []TrafficBucket  `json:"series"`
}

// DashboardSummary 只统计当前登录用户自己的套餐与规则用量。
func (s *Service) DashboardSummary(ctx context.Context, actor *Principal) (*DashboardSummary, error) {
	owner := actor.User.ID
	out := DashboardSummary{Series: []TrafficBucket{}}
	out.Package.Period = actor.User.PackagePeriod
	out.Package.RuleRemaining = -1
	out.Package.TrafficRemainByte = -1

	if actor.User.PackageID != nil && *actor.User.PackageID > 0 {
		var pkg model.Package
		if err := s.db.WithContext(ctx).First(&pkg, *actor.User.PackageID).Error; err == nil {
			out.Package.Name = pkg.Name
			out.Package.RuleLimit = pkg.RuleLimit
			out.Package.TrafficLimitGB = pkg.TrafficLimitGB

			var ruleUsed int64
			if err := s.db.WithContext(ctx).Model(&model.PortRule{}).
				Where("owner_user_id = ? AND direction = ?", owner, model.RuleDirectionEntry).
				Count(&ruleUsed).Error; err != nil {
				return nil, err
			}
			out.Package.RuleUsed = int(ruleUsed)
			if pkg.RuleLimit > 0 {
				remaining := pkg.RuleLimit - out.Package.RuleUsed
				if remaining < 0 {
					remaining = 0
				}
				out.Package.RuleRemaining = remaining
			}

			used, err := s.packageUsedBytes(ctx, pkg.ID)
			if err != nil {
				return nil, err
			}
			out.Package.TrafficUsedByte = used
			if pkg.TrafficLimitGB > 0 {
				limit := int64(pkg.TrafficLimitGB * 1024 * 1024 * 1024)
				out.Package.TrafficLimitByte = limit
				remaining := limit - used
				if remaining < 0 {
					remaining = 0
				}
				out.Package.TrafficRemainByte = remaining
			}
		}
	}

	ruleIDs := s.db.WithContext(ctx).Model(&model.PortRule{}).
		Select("id").Where("owner_user_id = ?", owner)

	from := time.Now().Add(-24 * time.Hour)
	statScope := s.db.WithContext(ctx).Model(&model.TrafficStat{}).
		Where("recorded_at >= ? AND rule_id IN (?)", from, ruleIDs)

	var totals struct {
		Up   int64
		Down int64
		Bill int64
	}
	if err := statScope.
		Select("COALESCE(SUM(up_bytes),0) AS up, COALESCE(SUM(down_bytes),0) AS down, COALESCE(SUM(bill_bytes),0) AS bill").
		Scan(&totals).Error; err != nil {
		return nil, err
	}
	out.Traffic24h = TrafficTotals{UpBytes: totals.Up, DownBytes: totals.Down, BillBytes: totals.Bill}

	var buckets []TrafficBucket
	if err := s.db.WithContext(ctx).Model(&model.TrafficStat{}).
		Where("recorded_at >= ? AND rule_id IN (?)", from, ruleIDs).
		Select("DATE_FORMAT(recorded_at, '%Y-%m-%d %H:00:00') AS bucket, " +
			"COALESCE(SUM(up_bytes),0) AS up_bytes, COALESCE(SUM(down_bytes),0) AS down_bytes, COALESCE(SUM(bill_bytes),0) AS bill_bytes").
		Group("bucket").Order("bucket asc").Scan(&buckets).Error; err != nil {
		return nil, err
	}
	if buckets == nil {
		buckets = []TrafficBucket{}
	}
	out.Series = buckets
	return &out, nil
}

// packageUsedBytes 汇总套餐内入口规则的累计计费流量，与配额强制使用的口径一致。
func (s *Service) packageUsedBytes(ctx context.Context, pkgID uint) (int64, error) {
	var used int64
	if err := s.db.WithContext(ctx).Model(&model.PortRule{}).
		Where("package_id = ? AND direction = ?", pkgID, model.RuleDirectionEntry).
		Select("COALESCE(SUM(used_bytes),0)").Scan(&used).Error; err != nil {
		return 0, err
	}
	return used, nil
}
