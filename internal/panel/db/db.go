package db

import (
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"aoe-tunnel/internal/panel/model"
)

func Open(dsn string, log *zap.Logger) (*gorm.DB, error) {
	gdb, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:  logger.Default.LogMode(logger.Warn),
		NowFunc: func() time.Time { return time.Now().Local() },
	})
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db: %w", err)
	}
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	if err := gdb.AutoMigrate(
		&model.User{},
		&model.Role{},
		&model.Permission{},
		&model.Session{},
		&model.Node{},
		&model.NodeInstance{},
		&model.Package{},
		&model.PortRule{},
		&model.TrafficStat{},
		&model.SiteConfig{},
		&model.PaymentChannel{},
		&model.PaymentOrder{},
		&model.CommissionRecord{},
	); err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}

	if err := migratePortRules(gdb, log); err != nil {
		return nil, fmt.Errorf("migrate port rules: %w", err)
	}

	// GORM 对连续大写字段会推断出 rate_up_mb_ps 这类列名，历史误建列需清理，统一用显式列名。
	for _, col := range []string{"rate_up_mb_ps", "rate_down_mb_ps"} {
		if !gdb.Migrator().HasColumn(&model.Node{}, col) {
			continue
		}
		if err := gdb.Migrator().DropColumn(&model.Node{}, col); err != nil {
			return nil, fmt.Errorf("drop legacy node column %s: %w", col, err)
		}
	}

	if err := migrateSyntheticExits(gdb, log); err != nil {
		return nil, fmt.Errorf("migrate synthetic exit rules: %w", err)
	}

	log.Info("mysql connected and schema migrated")
	return gdb, nil
}

// legacyPortRule 是链式时代 port_rules 的旧列投影，仅用于迁移读取。
type legacyPortRule struct {
	ID          uint
	Name        string
	Protocol    string
	EntryNode   string `gorm:"column:entry_node"`
	ExitNode    string `gorm:"column:exit_node"`
	ListenPort  int
	TargetAddr  string
	RatioUp     float64
	RatioDown   float64
	RateUpBps   int64
	RateDownBps int64
	QuotaBytes  int64
	UsedBytes   int64
	Enabled     bool
	Status      string
	Revision    int
	OwnerUserID *uint
}

// migratePortRules 把「入口+出口+链路」的旧规则拆成 entry/exit 两条规则并各自绑定套餐，
// 全部转换成功后再删除旧列；以 package_id=0 作为未转换标记，保证中途失败可幂等重跑。
func migratePortRules(gdb *gorm.DB, log *zap.Logger) error {
	if !gdb.Migrator().HasColumn(&model.PortRule{}, "entry_node") {
		return nil
	}

	var legacy []legacyPortRule
	if err := gdb.Table("port_rules").Where("package_id = 0").Order("id asc").Find(&legacy).Error; err != nil {
		return err
	}

	// 旧唯一索引 uk_entry_listen 约束 (entry_node, protocol, listen_port)，而迁移出的出口规则
	// 监听端口固定为 0 且 entry_node 为空占位，会撞唯一键，因此必须先于数据转换删除。
	if gdb.Migrator().HasIndex(&model.PortRule{}, "uk_entry_listen") {
		if err := gdb.Migrator().DropIndex(&model.PortRule{}, "uk_entry_listen"); err != nil {
			return fmt.Errorf("drop legacy index uk_entry_listen: %w", err)
		}
	}

	if len(legacy) > 0 {
		err := gdb.Transaction(func(tx *gorm.DB) error {
			for _, r := range legacy {
				entryNode := strings.TrimSpace(r.EntryNode)
				exitNode := strings.TrimSpace(r.ExitNode)
				if exitNode == "" {
					exitNode = entryNode
				}
				pkg := model.Package{
					Name:         r.Name,
					Description:  "由历史端口规则自动迁移",
					IngressNodes: uniqueCodes(entryNode),
					EgressNodes:  uniqueCodes(exitNode),
					Enabled:      true,
				}
				if err := tx.Create(&pkg).Error; err != nil {
					return err
				}

				if err := tx.Model(&model.PortRule{}).Where("id = ?", r.ID).
					Updates(map[string]any{
						"direction":   model.RuleDirectionEntry,
						"package_id":  pkg.ID,
						"node":        entryNode,
						"target_addr": "",
					}).Error; err != nil {
					return err
				}

				// 旧 NOT NULL 列在删除前仍需满足约束，新插入的出口规则填空字符串占位。
				now := time.Now()
				exitRow := map[string]any{
					"name":           r.Name + "-exit",
					"direction":      model.RuleDirectionExit,
					"package_id":     pkg.ID,
					"node":           exitNode,
					"protocol":       r.Protocol,
					"listen_port":    0,
					"target_addr":    r.TargetAddr,
					"entry_node":     "",
					"exit_node":      "",
					"chain_node_ids": "[]",
					"owner_user_id":  r.OwnerUserID,
					"ratio_up":       r.RatioUp,
					"ratio_down":     r.RatioDown,
					"rate_up_bps":    r.RateUpBps,
					"rate_down_bps":  r.RateDownBps,
					"quota_bytes":    r.QuotaBytes,
					"used_bytes":     r.UsedBytes,
					"enabled":        r.Enabled,
					"status":         r.Status,
					"revision":       r.Revision,
					"created_at":     now,
					"updated_at":     now,
				}
				if err := tx.Table("port_rules").Create(exitRow).Error; err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
		log.Info("legacy port rules migrated", zap.Int("count", len(legacy)))
	}

	for _, col := range []string{"chain_node_ids", "entry_node", "exit_node"} {
		if !gdb.Migrator().HasColumn(&model.PortRule{}, col) {
			continue
		}
		if err := gdb.Migrator().DropColumn(&model.PortRule{}, col); err != nil {
			return fmt.Errorf("drop column %s: %w", col, err)
		}
	}
	return nil
}

// migrateSyntheticExits 把历史落库的出口规则收回成合成规则：先用出口规则回填入口规则的目标地址，
// 再删除全部 direction=exit 行。以是否存在出口行作为幂等条件，可反复执行。
func migrateSyntheticExits(gdb *gorm.DB, log *zap.Logger) error {
	var count int64
	if err := gdb.Model(&model.PortRule{}).
		Where("direction = ?", model.RuleDirectionExit).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return nil
	}

	var exits []model.PortRule
	if err := gdb.Where("direction = ?", model.RuleDirectionExit).
		Order("id asc").Find(&exits).Error; err != nil {
		return err
	}

	err := gdb.Transaction(func(tx *gorm.DB) error {
		for _, exit := range exits {
			target := strings.TrimSpace(exit.TargetAddr)
			if target == "" || exit.PackageID == 0 {
				continue
			}
			if err := tx.Model(&model.PortRule{}).
				Where("package_id = ? AND direction = ? AND target_addr = ''",
					exit.PackageID, model.RuleDirectionEntry).
				Update("target_addr", target).Error; err != nil {
				return err
			}
		}
		return tx.Where("direction = ?", model.RuleDirectionExit).Delete(&model.PortRule{}).Error
	})
	if err != nil {
		return err
	}
	log.Info("synthetic exit rules folded into entry rules", zap.Int64("removed", count))
	return nil
}

func uniqueCodes(codes ...string) []string {
	out := make([]string, 0, len(codes))
	seen := make(map[string]struct{}, len(codes))
	for _, c := range codes {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		out = append(out, c)
	}
	return out
}
