package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"gorm.io/gorm"

	"aoe-tunnel/internal/panel/model"
)

// normalizeHTTPURL 校验地址为 http(s)://host 形式并去掉末尾斜杠；空值表示不配置。
func normalizeHTTPURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", fmt.Errorf("地址需为 http(s)://host 形式: %s", raw)
	}
	return strings.TrimSuffix(raw, "/"), nil
}

// normalizeBroker 校验 MQTT 地址协议头，避免把 HTTP 地址误填进来。
func normalizeBroker(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	for _, p := range []string{"ssl://", "tcp://", "ws://", "wss://"} {
		if strings.HasPrefix(raw, p) {
			return raw, nil
		}
	}
	return "", fmt.Errorf("MQTT 地址需以 ssl://、tcp://、ws:// 或 wss:// 开头: %s", raw)
}

// siteConfigID 站点配置固定单行，避免为每个字段单独建表。
const siteConfigID uint = 1

// PublicSiteConfig 是登录/注册页可见的站点信息，不暴露分销等后台配置。
type PublicSiteConfig struct {
	SiteTitle      string `json:"site_title"`
	SiteSubtitle   string `json:"site_subtitle"`
	Announcement   string `json:"announcement"`
	RegisterOpen   bool   `json:"register_open"`
	RegisterReview bool   `json:"register_review"`
}

type UpdateSystemSettingsInput struct {
	SiteTitle         *string  `json:"site_title,omitempty" binding:"omitempty,max=128"`
	SiteSubtitle      *string  `json:"site_subtitle,omitempty" binding:"omitempty,max=255"`
	Announcement      *string  `json:"announcement,omitempty" binding:"omitempty,max=512"`
	RegisterOpen      *bool    `json:"register_open,omitempty"`
	RegisterReview    *bool    `json:"register_review,omitempty"`
	DistributionOpen  *bool    `json:"distribution_open,omitempty"`
	DistributionRatio *float64 `json:"distribution_ratio,omitempty"`
	// 站点公网地址与 Agent 下发地址：留空表示回退环境变量或请求头自动拼接。
	PublicURL        *string `json:"public_url,omitempty" binding:"omitempty,max=255"`
	AgentDownloadURL *string `json:"agent_download_url,omitempty" binding:"omitempty,max=255"`
	AgentBroker      *string `json:"agent_broker,omitempty" binding:"omitempty,max=255"`
}

// SiteConfig 读取站点配置，首次访问时写入默认值。
func (s *Service) SiteConfig(ctx context.Context) (*model.SiteConfig, error) {
	var cfg model.SiteConfig
	err := s.db.WithContext(ctx).First(&cfg, siteConfigID).Error
	if err == nil {
		return &cfg, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	cfg = model.SiteConfig{
		ID:             siteConfigID,
		SiteTitle:      "Aoe-tunnel",
		SiteSubtitle:   "控制台入口",
		RegisterOpen:   false,
		RegisterReview: true,
	}
	if err := s.db.WithContext(ctx).Create(&cfg).Error; err != nil {
		// 并发首次访问可能撞主键，重新读取即可。
		if loadErr := s.db.WithContext(ctx).First(&cfg, siteConfigID).Error; loadErr != nil {
			return nil, err
		}
	}
	return &cfg, nil
}

func (s *Service) PublicSiteConfig(ctx context.Context) (*PublicSiteConfig, error) {
	cfg, err := s.SiteConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &PublicSiteConfig{
		SiteTitle:      cfg.SiteTitle,
		SiteSubtitle:   cfg.SiteSubtitle,
		Announcement:   cfg.Announcement,
		RegisterOpen:   cfg.RegisterOpen,
		RegisterReview: cfg.RegisterReview,
	}, nil
}

func (s *Service) UpdateSystemSettings(ctx context.Context, in UpdateSystemSettingsInput) (*model.SiteConfig, error) {
	cfg, err := s.SiteConfig(ctx)
	if err != nil {
		return nil, err
	}
	if in.SiteTitle != nil {
		title := strings.TrimSpace(*in.SiteTitle)
		if title == "" {
			return nil, fmt.Errorf("site_title required")
		}
		cfg.SiteTitle = title
	}
	if in.SiteSubtitle != nil {
		cfg.SiteSubtitle = strings.TrimSpace(*in.SiteSubtitle)
	}
	if in.Announcement != nil {
		cfg.Announcement = strings.TrimSpace(*in.Announcement)
	}
	if in.RegisterOpen != nil {
		cfg.RegisterOpen = *in.RegisterOpen
	}
	if in.RegisterReview != nil {
		cfg.RegisterReview = *in.RegisterReview
	}
	if in.DistributionOpen != nil {
		cfg.DistributionOpen = *in.DistributionOpen
	}
	if in.DistributionRatio != nil {
		ratio := *in.DistributionRatio
		if ratio < 0 || ratio > 100 {
			return nil, fmt.Errorf("distribution_ratio must be between 0 and 100")
		}
		cfg.DistributionRatio = ratio
	}
	if in.PublicURL != nil {
		v, err := normalizeHTTPURL(*in.PublicURL)
		if err != nil {
			return nil, err
		}
		cfg.PublicURL = v
	}
	if in.AgentDownloadURL != nil {
		v, err := normalizeHTTPURL(*in.AgentDownloadURL)
		if err != nil {
			return nil, err
		}
		cfg.AgentDownloadURL = v
	}
	if in.AgentBroker != nil {
		v, err := normalizeBroker(*in.AgentBroker)
		if err != nil {
			return nil, err
		}
		cfg.AgentBroker = v
	}
	if err := s.db.WithContext(ctx).Save(cfg).Error; err != nil {
		return nil, err
	}
	return cfg, nil
}
