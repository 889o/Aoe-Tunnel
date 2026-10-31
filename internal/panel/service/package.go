package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"gorm.io/gorm"

	"aoe-tunnel/internal/panel/model"
)

// CreatePackageInput 绑定入口与出口节点；同一套餐内的入口规则自动向所有出口扇出。
type CreatePackageInput struct {
	Name           string             `json:"name" binding:"required,max=128"`
	Description    string             `json:"description,omitempty" binding:"max=255"`
	IngressNodes   []string           `json:"ingress_nodes"`
	EgressNodes    []string           `json:"egress_nodes"`
	TrafficLimitGB float64            `json:"traffic_limit_gb"`
	RuleLimit      int                `json:"rule_limit"`
	Prices         map[string]float64 `json:"prices"`
	Enabled        *bool              `json:"enabled,omitempty"`
}

type UpdatePackageInput struct {
	Name           *string             `json:"name,omitempty" binding:"omitempty,max=128"`
	Description    *string             `json:"description,omitempty" binding:"omitempty,max=255"`
	IngressNodes   *[]string           `json:"ingress_nodes,omitempty"`
	EgressNodes    *[]string           `json:"egress_nodes,omitempty"`
	TrafficLimitGB *float64            `json:"traffic_limit_gb,omitempty"`
	RuleLimit      *int                `json:"rule_limit,omitempty"`
	Prices         *map[string]float64 `json:"prices,omitempty"`
	Enabled        *bool               `json:"enabled,omitempty"`
}

func (s *Service) getPackage(ctx context.Context, id uint) (*model.Package, error) {
	var pkg model.Package
	if err := s.db.WithContext(ctx).First(&pkg, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	normalizePackageSlices(&pkg)
	pkg.NormalizePackagePrices()
	return &pkg, nil
}

func (s *Service) ListPackages(ctx context.Context) ([]model.Package, error) {
	var pkgs []model.Package
	if err := s.db.WithContext(ctx).Order("id asc").Find(&pkgs).Error; err != nil {
		return nil, err
	}
	for i := range pkgs {
		normalizePackageSlices(&pkgs[i])
		pkgs[i].NormalizePackagePrices()
	}
	return pkgs, nil
}

func (s *Service) CreatePackage(ctx context.Context, in CreatePackageInput) (*model.Package, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, fmt.Errorf("package name required")
	}
	ingress, err := s.normalizePackageNodes(ctx, in.IngressNodes, model.NodeTypeIngress, "ingress")
	if err != nil {
		return nil, err
	}
	egress, err := s.normalizePackageNodes(ctx, in.EgressNodes, model.NodeTypeEgress, "egress")
	if err != nil {
		return nil, err
	}
	if in.TrafficLimitGB < 0 {
		return nil, fmt.Errorf("traffic_limit_gb must not be negative")
	}
	if in.RuleLimit < 0 {
		return nil, fmt.Errorf("rule_limit must not be negative")
	}
	prices, err := normalizePackagePrices(in.Prices)
	if err != nil {
		return nil, err
	}
	pkg := &model.Package{
		Name:           name,
		Description:    strings.TrimSpace(in.Description),
		IngressNodes:   ingress,
		EgressNodes:    egress,
		TrafficLimitGB: in.TrafficLimitGB,
		RuleLimit:      in.RuleLimit,
		Prices:         prices,
		Enabled:        true,
	}
	if in.Enabled != nil {
		pkg.Enabled = *in.Enabled
	}
	if err := s.db.WithContext(ctx).Create(pkg).Error; err != nil {
		return nil, err
	}
	return pkg, nil
}

func (s *Service) UpdatePackage(ctx context.Context, id uint, in UpdatePackageInput) (*model.Package, error) {
	pkg, err := s.getPackage(ctx, id)
	if err != nil {
		return nil, err
	}
	oldEgress := append([]string(nil), pkg.EgressNodes...)
	oldEnabled := pkg.Enabled

	if in.Name != nil {
		if name := strings.TrimSpace(*in.Name); name != "" {
			pkg.Name = name
		}
	}
	if in.Description != nil {
		pkg.Description = strings.TrimSpace(*in.Description)
	}
	if in.IngressNodes != nil {
		nodes, err := s.normalizePackageNodes(ctx, *in.IngressNodes, model.NodeTypeIngress, "ingress")
		if err != nil {
			return nil, err
		}
		pkg.IngressNodes = nodes
	}
	if in.EgressNodes != nil {
		nodes, err := s.normalizePackageNodes(ctx, *in.EgressNodes, model.NodeTypeEgress, "egress")
		if err != nil {
			return nil, err
		}
		pkg.EgressNodes = nodes
	}
	if in.TrafficLimitGB != nil {
		if *in.TrafficLimitGB < 0 {
			return nil, fmt.Errorf("traffic_limit_gb must not be negative")
		}
		pkg.TrafficLimitGB = *in.TrafficLimitGB
	}
	if in.RuleLimit != nil {
		if *in.RuleLimit < 0 {
			return nil, fmt.Errorf("rule_limit must not be negative")
		}
		pkg.RuleLimit = *in.RuleLimit
	}
	if in.Prices != nil {
		prices, err := normalizePackagePrices(*in.Prices)
		if err != nil {
			return nil, err
		}
		pkg.Prices = prices
	}
	if in.Enabled != nil {
		pkg.Enabled = *in.Enabled
	}
	if err := s.db.WithContext(ctx).Save(pkg).Error; err != nil {
		return nil, err
	}

	// 出口集合或启用状态变化会影响入口扇出列表，需要升版重推入口规则；
	// 被移除或停用的出口节点还需清理已下发的合成出口规则。
	if !sameCodes(oldEgress, pkg.EgressNodes) || oldEnabled != pkg.Enabled {
		removed := diffCodes(oldEgress, pkg.EgressNodes)
		if oldEnabled && !pkg.Enabled {
			removed = oldEgress
		}
		s.purgePackageExitRules(ctx, pkg.ID, removed)
		s.redispatchPackageEntries(ctx, pkg.ID)
	}
	return pkg, nil
}

func (s *Service) DeletePackage(ctx context.Context, id uint) error {
	if _, err := s.getPackage(ctx, id); err != nil {
		return err
	}
	var count int64
	if err := s.db.WithContext(ctx).Model(&model.PortRule{}).Where("package_id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("%w: package still has %d port rules", ErrConflict, count)
	}
	return s.db.WithContext(ctx).Delete(&model.Package{}, id).Error
}

// normalizePackageNodes 去空、去重，并校验节点存在且类型与绑定槽位一致。
func (s *Service) normalizePackageNodes(ctx context.Context, codes []string, wantType int, field string) ([]string, error) {
	out := make([]string, 0, len(codes))
	seen := make(map[string]struct{}, len(codes))
	for _, raw := range codes {
		code := strings.TrimSpace(raw)
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		node, err := s.getNode(ctx, code)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return nil, fmt.Errorf("%s node %q: %w", field, code, ErrNotFound)
			}
			return nil, err
		}
		if node.Type != wantType {
			return nil, fmt.Errorf("%w: node %s is not %s type", ErrConflict, code, field)
		}
		seen[code] = struct{}{}
		out = append(out, code)
	}
	return out, nil
}

func normalizePackageSlices(pkg *model.Package) {
	if pkg.IngressNodes == nil {
		pkg.IngressNodes = []string{}
	}
	if pkg.EgressNodes == nil {
		pkg.EgressNodes = []string{}
	}
}

// normalizePackagePrices 清洗周期价格：key 必须是合法周期，金额非负且至少选择一个周期。
func normalizePackagePrices(in map[string]float64) (map[string]float64, error) {
	out := make(map[string]float64, len(in))
	for rawKey, price := range in {
		key := strings.TrimSpace(rawKey)
		if !model.ValidPackageDuration(key) {
			return nil, fmt.Errorf("invalid billing period %q", rawKey)
		}
		if price < 0 {
			return nil, fmt.Errorf("price for %s must not be negative", key)
		}
		out[key] = math.Round(price*100) / 100
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("at least one billing period is required")
	}
	return out, nil
}

func sameCodes(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// diffCodes 返回在旧集合中但不在新集合中的编码。
func diffCodes(old, cur []string) []string {
	out := make([]string, 0, len(old))
	for _, c := range old {
		if !containsCode(cur, c) {
			out = append(out, c)
		}
	}
	return out
}
