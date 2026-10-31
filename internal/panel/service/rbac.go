package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"aoe-tunnel/internal/panel/model"
)

// 权限码统一使用「资源:动作」，前端菜单与按钮显隐都按这些码判定。
const (
	PermDashboardView = "dashboard:view"

	PermNodeRead   = "node:read"
	PermNodeCreate = "node:create"
	PermNodeUpdate = "node:update"
	PermNodeDelete = "node:delete"
	PermNodeReport = "node:report"

	PermPackageList   = "package:list"
	PermPackageCreate = "package:create"
	PermPackageUpdate = "package:update"
	PermPackageDelete = "package:delete"

	PermPortList      = "port:list"
	PermPortRead      = "port:read"
	PermPortCreate    = "port:create"
	PermPortUpdate    = "port:update"
	PermPortDelete    = "port:delete"
	PermPortStats     = "port:stats"
	PermPortManageAll = "port:manage-all"

	PermUserList   = "user:list"
	PermUserCreate = "user:create"
	PermUserUpdate = "user:update"
	PermUserDelete = "user:delete"

	PermRoleList   = "role:list"
	PermRoleCreate = "role:create"
	PermRoleUpdate = "role:update"
	PermRoleDelete = "role:delete"

	PermSessionList      = "session:list"
	PermSessionKick      = "session:kick"
	PermSessionManageAll = "session:manage-all"

	PermProfileView   = "profile:view"
	PermProfileUpdate = "profile:update"

	// 钱包商城：查看余额与购买套餐，普通用户默认可用。
	PermWalletView = "wallet:view"

	PermSystemView   = "system:view"
	PermSystemUpdate = "system:update"
)

// 内置角色编码。
const (
	RoleSuperAdmin = "super_admin"
	RoleAdmin      = "admin"
	RoleUser       = "user"
	RoleAgent      = "agent"
)

// PermissionCatalog 是全部权限码的字典，启动时同步进 permissions 表供前端渲染勾选矩阵。
var PermissionCatalog = []model.Permission{
	{Code: PermDashboardView, Name: "查看概览", Group: "dashboard", Sort: 1},

	{Code: PermNodeRead, Name: "查看节点", Group: "node", Sort: 1},
	{Code: PermNodeCreate, Name: "创建节点", Group: "node", Sort: 2},
	{Code: PermNodeUpdate, Name: "编辑节点", Group: "node", Sort: 3},
	{Code: PermNodeDelete, Name: "删除节点", Group: "node", Sort: 4},
	{Code: PermNodeReport, Name: "节点上报（Agent 专用）", Group: "node", Sort: 5},

	{Code: PermPackageList, Name: "查看套餐", Group: "package", Sort: 1},
	{Code: PermPackageCreate, Name: "创建套餐", Group: "package", Sort: 2},
	{Code: PermPackageUpdate, Name: "编辑套餐", Group: "package", Sort: 3},
	{Code: PermPackageDelete, Name: "删除套餐", Group: "package", Sort: 4},

	{Code: PermPortList, Name: "列出端口", Group: "port", Sort: 1},
	{Code: PermPortRead, Name: "查看端口", Group: "port", Sort: 2},
	{Code: PermPortCreate, Name: "创建端口", Group: "port", Sort: 3},
	{Code: PermPortUpdate, Name: "编辑端口", Group: "port", Sort: 4},
	{Code: PermPortDelete, Name: "删除端口", Group: "port", Sort: 5},
	{Code: PermPortStats, Name: "查看端口流量", Group: "port", Sort: 6},
	{Code: PermPortManageAll, Name: "管理全部端口（跨用户）", Group: "port", Sort: 7},

	{Code: PermUserList, Name: "查看用户", Group: "user", Sort: 1},
	{Code: PermUserCreate, Name: "创建用户", Group: "user", Sort: 2},
	{Code: PermUserUpdate, Name: "编辑用户", Group: "user", Sort: 3},
	{Code: PermUserDelete, Name: "删除用户", Group: "user", Sort: 4},

	{Code: PermRoleList, Name: "查看角色", Group: "role", Sort: 1},
	{Code: PermRoleCreate, Name: "创建角色", Group: "role", Sort: 2},
	{Code: PermRoleUpdate, Name: "编辑角色权限", Group: "role", Sort: 3},
	{Code: PermRoleDelete, Name: "删除角色", Group: "role", Sort: 4},

	{Code: PermSessionList, Name: "查看会话", Group: "session", Sort: 1},
	{Code: PermSessionKick, Name: "踢下线", Group: "session", Sort: 2},
	{Code: PermSessionManageAll, Name: "管理全部会话", Group: "session", Sort: 3},

	{Code: PermProfileView, Name: "查看个人设置", Group: "profile", Sort: 1},
	{Code: PermProfileUpdate, Name: "修改个人设置", Group: "profile", Sort: 2},

	{Code: PermWalletView, Name: "钱包商城", Group: "wallet", Sort: 1},

	{Code: PermSystemView, Name: "查看系统设置", Group: "system", Sort: 1},
	{Code: PermSystemUpdate, Name: "修改系统设置", Group: "system", Sort: 2},
}

// BuiltinRoles 定义三个内置角色的权限边界，启动时按此同步，禁止页面修改。
var BuiltinRoles = []model.Role{
	{
		Code:        RoleSuperAdmin,
		Name:        "超级管理员",
		Description: "拥有全部权限，内置角色不可删除或修改",
		Builtin:     true,
	},
	{
		Code:        RoleAdmin,
		Name:        "管理员",
		Description: "管理节点、端口与用户，但不能创建用户或修改角色权限",
		Builtin:     true,
	},
	{
		Code:        RoleUser,
		Name:        "普通用户",
		Description: "仅管理自己创建的端口，可查看节点与个人设置",
		Builtin:     true,
	},
	{
		Code:        RoleAgent,
		Name:        "Agent 节点",
		Description: "内置 Agent 账号专用：只能查看节点目录并上报运行数据",
		Builtin:     true,
	},
}

func allPermissionCodes() []string {
	out := make([]string, 0, len(PermissionCatalog))
	for _, p := range PermissionCatalog {
		out = append(out, p.Code)
	}
	return out
}

func builtinRolePermissions(code string) []string {
	switch code {
	case RoleSuperAdmin:
		return allPermissionCodes()
	case RoleAdmin:
		return []string{
			PermDashboardView,
			PermNodeRead, PermNodeCreate, PermNodeUpdate, PermNodeDelete,
			PermPackageList, PermPackageCreate, PermPackageUpdate, PermPackageDelete,
			PermPortList, PermPortRead, PermPortCreate, PermPortUpdate, PermPortDelete, PermPortStats, PermPortManageAll,
			PermUserList, PermUserUpdate,
			PermRoleList,
			PermSessionList, PermSessionKick, PermSessionManageAll,
			PermProfileView, PermProfileUpdate,
			PermWalletView,
			PermSystemView,
		}
	case RoleAgent:
		return []string{PermNodeRead, PermNodeReport}
	default: // RoleUser
		return []string{
			PermDashboardView,
			PermNodeRead,
			PermPackageList,
			PermPortList, PermPortRead, PermPortCreate, PermPortUpdate, PermPortDelete, PermPortStats,
			PermProfileView, PermProfileUpdate,
			PermWalletView,
		}
	}
}

var roleCodeRe = regexp.MustCompile(`^[a-z][a-z0-9_]{1,31}$`)

// MenuItem 是后端下发的动态菜单节点，前端按 icon 名称映射 lucide 图标。
type MenuItem struct {
	Key      string     `json:"key"`
	Path     string     `json:"path"`
	Title    string     `json:"title"`
	Icon     string     `json:"icon"`
	Perm     string     `json:"perm"`
	Children []MenuItem `json:"children,omitempty"`
}

var allMenus = []MenuItem{
	{Key: "dashboard", Path: "/dashboard", Title: "Dashboard", Icon: "LayoutDashboard", Perm: PermDashboardView},
	{Key: "nodes", Path: "/nodes", Title: "节点设置", Icon: "Network", Perm: PermNodeRead},
	{Key: "ports", Path: "/ports", Title: "转发规则", Icon: "Boxes", Perm: PermPortList},
	{Key: "packages", Path: "/packages", Title: "套餐管理", Icon: "Package", Perm: PermPackageList},
	{Key: "wallet", Path: "/store", Title: "钱包商城", Icon: "Wallet", Perm: PermWalletView},
	{Key: "monitor", Path: "/monitor", Title: "服务器监控", Icon: "Cpu", Perm: PermNodeRead},
	{Key: "users", Path: "/users", Title: "用户管理", Icon: "Users", Perm: PermUserList},
	{Key: "profile", Path: "/profile", Title: "个人设置", Icon: "UserCog", Perm: PermProfileView},
	{Key: "settings", Path: "/settings", Title: "系统设置", Icon: "Settings", Perm: PermSystemView},
}

// adminOnlyMenus 是仅管理员可见的菜单：普通用户即使有查看权限也不展示入口。
var adminOnlyMenus = map[string]bool{
	"nodes":    true,
	"packages": true,
	"users":    true,
	"settings": true,
}

// MenusFor 按权限过滤菜单，保持固定顺序：Dashboard、节点设置、转发规则、套餐管理、钱包商城、服务器监控、用户管理、个人设置、系统设置。
func MenusFor(perms map[string]struct{}, roleCode string) []MenuItem {
	isAdmin := roleCode == RoleSuperAdmin || roleCode == RoleAdmin
	out := make([]MenuItem, 0, len(allMenus))
	for _, m := range allMenus {
		if _, ok := perms[m.Perm]; !ok {
			continue
		}
		if adminOnlyMenus[m.Key] && !isAdmin {
			continue
		}
		out = append(out, m)
	}
	return out
}

// PermissionSet 把权限码数组转成集合，便于 O(1) 判定。
func PermissionSet(codes []string) map[string]struct{} {
	set := make(map[string]struct{}, len(codes))
	for _, c := range codes {
		set[c] = struct{}{}
	}
	return set
}

// Bootstrap 在每次面板启动时同步权限字典与内置角色，并保证存在可登录的超级管理员与 Agent 账号。
func (s *Service) Bootstrap(ctx context.Context, adminUser, adminPass, agentUser, agentPass string) error {
	for _, p := range PermissionCatalog {
		row := model.Permission{Code: p.Code}
		if err := s.db.WithContext(ctx).Where("code = ?", p.Code).
			Assign(model.Permission{Name: p.Name, Group: p.Group, Sort: p.Sort}).
			FirstOrCreate(&row).Error; err != nil {
			return fmt.Errorf("bootstrap permission %s: %w", p.Code, err)
		}
	}

	roles := make(map[string]model.Role, len(BuiltinRoles))
	for _, r := range BuiltinRoles {
		row := model.Role{Code: r.Code}
		if err := s.db.WithContext(ctx).Where("code = ?", r.Code).
			Assign(model.Role{
				Name:            r.Name,
				Description:     r.Description,
				Builtin:         true,
				PermissionCodes: builtinRolePermissions(r.Code),
			}).FirstOrCreate(&row).Error; err != nil {
			return fmt.Errorf("bootstrap role %s: %w", r.Code, err)
		}
		roles[r.Code] = row
	}

	// 存量 users.role 字符串回填 role_id；未知取值归为普通用户。
	var legacy []model.User
	if err := s.db.WithContext(ctx).Where("role_id = 0").Find(&legacy).Error; err != nil {
		return fmt.Errorf("bootstrap backfill users: %w", err)
	}
	for _, u := range legacy {
		code := strings.TrimSpace(strings.ToLower(u.Role))
		role, ok := roles[code]
		if !ok {
			role = roles[RoleUser]
		}
		if err := s.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", u.ID).
			Update("role_id", role.ID).Error; err != nil {
			return fmt.Errorf("bootstrap backfill user %d: %w", u.ID, err)
		}
	}

	var admin model.User
	err := s.db.WithContext(ctx).Where("username = ?", adminUser).First(&admin).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		hash, hErr := hashPassword(adminPass)
		if hErr != nil {
			return hErr
		}
		admin = model.User{
			Username:     adminUser,
			PasswordHash: hash,
			DisplayName:  "超级管理员",
			Role:         RoleSuperAdmin,
			RoleID:       roles[RoleSuperAdmin].ID,
			Status:       "active",
		}
		if err := s.db.WithContext(ctx).Create(&admin).Error; err != nil {
			return fmt.Errorf("bootstrap admin: %w", err)
		}
		s.log.Warn("bootstrap admin account created",
			zap.String("username", adminUser), zap.String("role", RoleSuperAdmin))
	} else if err != nil {
		return fmt.Errorf("bootstrap admin lookup: %w", err)
	}

	// Agent 内置账号只用于登录换取 JWT，角色固定为 agent，不参与 Web 管理。
	if agentUser != "" {
		var agent model.User
		err := s.db.WithContext(ctx).Where("username = ?", agentUser).First(&agent).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			hash, hErr := hashPassword(agentPass)
			if hErr != nil {
				return hErr
			}
			agent = model.User{
				Username:     agentUser,
				PasswordHash: hash,
				DisplayName:  "Agent 节点账号",
				Role:         RoleAgent,
				RoleID:       roles[RoleAgent].ID,
				Status:       "active",
			}
			if err := s.db.WithContext(ctx).Create(&agent).Error; err != nil {
				return fmt.Errorf("bootstrap agent: %w", err)
			}
			s.log.Warn("bootstrap agent account created",
				zap.String("username", agentUser), zap.String("role", RoleAgent))
		} else if err != nil {
			return fmt.Errorf("bootstrap agent lookup: %w", err)
		}
	}

	// 存量端口规则没有归属人，统一划归超级管理员，避免普通用户看到历史规则。
	if admin.ID != 0 {
		if err := s.db.WithContext(ctx).Model(&model.PortRule{}).
			Where("owner_user_id IS NULL").Update("owner_user_id", admin.ID).Error; err != nil {
			return fmt.Errorf("bootstrap backfill port rules: %w", err)
		}
	}

	// 存量节点回填 type=1 只是 DB 标记；Agent 未带 --ingress/--egress 时仍走旧逻辑。
	if err := s.db.WithContext(ctx).Model(&model.Node{}).
		Where("type = ?", model.NodeTypeLegacy).Update("type", model.NodeTypeIngress).Error; err != nil {
		return fmt.Errorf("bootstrap backfill node type: %w", err)
	}
	// 校正节点令牌哈希：密钥来源切换或历史数据漂移时，按当前 epoch 重算，保证安装脚本与上线校验一致。
	var nodes []model.Node
	if err := s.db.WithContext(ctx).Find(&nodes).Error; err != nil {
		return fmt.Errorf("bootstrap backfill node token: %w", err)
	}
	for _, n := range nodes {
		epoch := n.NodeTokenEpoch
		if epoch <= 0 {
			epoch = 1
		}
		hash := s.nodeTokenHash(n.Code, epoch)
		if hash == n.NodeTokenHash && epoch == n.NodeTokenEpoch {
			continue
		}
		if err := s.db.WithContext(ctx).Model(&model.Node{}).Where("id = ?", n.ID).
			Updates(map[string]any{
				"node_token_hash":  hash,
				"node_token_epoch": epoch,
			}).Error; err != nil {
			return fmt.Errorf("bootstrap backfill node token %s: %w", n.Code, err)
		}
	}
	return nil
}

// PermissionCodes 返回字典中的全部权限码，供角色创建/更新时校验。
func (s *Service) ListPermissions(ctx context.Context) ([]model.Permission, error) {
	var rows []model.Permission
	if err := s.db.WithContext(ctx).Order("`group` asc, `sort` asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

type RoleView struct {
	model.Role
	UserCount int64 `json:"user_count"`
}

func (s *Service) ListRoles(ctx context.Context) ([]RoleView, error) {
	var roles []model.Role
	if err := s.db.WithContext(ctx).Order("id asc").Find(&roles).Error; err != nil {
		return nil, err
	}
	out := make([]RoleView, 0, len(roles))
	for _, r := range roles {
		var count int64
		if err := s.db.WithContext(ctx).Model(&model.User{}).Where("role_id = ?", r.ID).Count(&count).Error; err != nil {
			return nil, err
		}
		if r.PermissionCodes == nil {
			r.PermissionCodes = []string{}
		}
		out = append(out, RoleView{Role: r, UserCount: count})
	}
	return out, nil
}

type CreateRoleInput struct {
	Code            string   `json:"code" binding:"required,max=32"`
	Name            string   `json:"name" binding:"required,max=64"`
	Description     string   `json:"description,omitempty" binding:"max=255"`
	PermissionCodes []string `json:"permission_codes"`
}

func (s *Service) CreateRole(ctx context.Context, in CreateRoleInput) (*model.Role, error) {
	code := strings.TrimSpace(in.Code)
	if !roleCodeRe.MatchString(code) {
		return nil, fmt.Errorf("role code must match ^[a-z][a-z0-9_]{1,31}$")
	}
	codes, err := normalizePermissionCodes(in.PermissionCodes)
	if err != nil {
		return nil, err
	}
	var count int64
	if err := s.db.WithContext(ctx).Model(&model.Role{}).Where("code = ?", code).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, fmt.Errorf("%w: role %s already exists", ErrConflict, code)
	}
	role := model.Role{
		Code:            code,
		Name:            strings.TrimSpace(in.Name),
		Description:     strings.TrimSpace(in.Description),
		Builtin:         false,
		PermissionCodes: codes,
	}
	if err := s.db.WithContext(ctx).Create(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

type UpdateRoleInput struct {
	Name            *string   `json:"name,omitempty" binding:"omitempty,max=64"`
	Description     *string   `json:"description,omitempty" binding:"omitempty,max=255"`
	PermissionCodes *[]string `json:"permission_codes,omitempty"`
}

func (s *Service) UpdateRole(ctx context.Context, id uint, in UpdateRoleInput) (*model.Role, error) {
	var role model.Role
	if err := s.db.WithContext(ctx).First(&role, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if role.Builtin {
		return nil, fmt.Errorf("%w: builtin role cannot be modified", ErrForbidden)
	}
	if in.Name != nil {
		role.Name = strings.TrimSpace(*in.Name)
	}
	if in.Description != nil {
		role.Description = strings.TrimSpace(*in.Description)
	}
	if in.PermissionCodes != nil {
		codes, err := normalizePermissionCodes(*in.PermissionCodes)
		if err != nil {
			return nil, err
		}
		role.PermissionCodes = codes
	}
	if err := s.db.WithContext(ctx).Save(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (s *Service) DeleteRole(ctx context.Context, id uint) error {
	var role model.Role
	if err := s.db.WithContext(ctx).First(&role, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	if role.Builtin {
		return fmt.Errorf("%w: builtin role cannot be deleted", ErrForbidden)
	}
	var count int64
	if err := s.db.WithContext(ctx).Model(&model.User{}).Where("role_id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("%w: role still has %d users", ErrConflict, count)
	}
	return s.db.WithContext(ctx).Delete(&role).Error
}

func (s *Service) roleByID(ctx context.Context, id uint) (*model.Role, error) {
	var role model.Role
	if err := s.db.WithContext(ctx).First(&role, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if role.PermissionCodes == nil {
		role.PermissionCodes = []string{}
	}
	return &role, nil
}

func (s *Service) roleByCode(ctx context.Context, code string) (*model.Role, error) {
	var role model.Role
	if err := s.db.WithContext(ctx).Where("code = ?", code).First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if role.PermissionCodes == nil {
		role.PermissionCodes = []string{}
	}
	return &role, nil
}

// normalizePermissionCodes 去重并校验权限码存在于字典，未知码直接拒绝。
func normalizePermissionCodes(in []string) ([]string, error) {
	valid := PermissionSet(allPermissionCodes())
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, raw := range in {
		code := strings.TrimSpace(raw)
		if code == "" {
			continue
		}
		if _, ok := valid[code]; !ok {
			return nil, fmt.Errorf("unknown permission code %q", code)
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		out = append(out, code)
	}
	return out, nil
}
