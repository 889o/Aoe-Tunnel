package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"aoe-tunnel/internal/panel/api"
	"aoe-tunnel/internal/panel/auth"
	"aoe-tunnel/internal/panel/config"
	"aoe-tunnel/internal/panel/db"
	"aoe-tunnel/internal/panel/service"
	"aoe-tunnel/pkg/logger"
	"aoe-tunnel/pkg/mqttcli"
	aoesignal "aoe-tunnel/pkg/signal"
	"aoe-tunnel/pkg/version"
)

// serveWeb 托管前端构建产物，非 /api 路径回退 index.html 以支持前端路由。
func serveWeb(r *gin.Engine, dir string) error {
	index := filepath.Join(dir, "index.html")
	if _, err := os.Stat(index); err != nil {
		return fmt.Errorf("index.html: %w", err)
	}
	r.Static("/assets", filepath.Join(dir, "assets"))
	r.StaticFile("/favicon.ico", filepath.Join(dir, "favicon.ico"))
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "not found"})
			return
		}
		// index.html 每次发版都会换 hash，禁止缓存避免浏览器继续加载旧页面。
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.File(index)
	})
	return nil
}

// agentDistFiles 是允许公网下载的安装物料白名单，避免把目录内其他文件暴露出去。
var agentDistFiles = map[string]string{
	"install.sh":             "install.sh",
	"install-default.sh":     "install-default.sh",
	"aoe-agent-linux-amd64":  "aoe-agent-linux-amd64",
	"aoe-agent-linux-arm64":  "aoe-agent-linux-arm64",
	"aoe-agent-certs.tar.gz": "aoe-agent-certs.tar.gz",
}

// serveDownloads 暴露 /downloads/:file，供远端服务器安装 Agent 时拉取二进制与证书。
func serveDownloads(router *gin.Engine, dir string) error {
	if _, err := os.Stat(dir); err != nil {
		return err
	}
	router.GET("/downloads/:file", func(c *gin.Context) {
		name, ok := agentDistFiles[c.Param("file")]
		if !ok {
			c.String(http.StatusNotFound, "not found")
			return
		}
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err != nil {
			c.String(http.StatusNotFound, "not found")
			return
		}
		c.Header("Content-Type", "application/octet-stream")
		c.File(path)
	})
	return nil
}

// redirectHTTPS 把 HTTP 请求永久跳转到 HTTPS；Host 去掉端口后重新拼装。
func redirectHTTPS(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	if i := strings.LastIndex(host, ":"); i > 0 {
		host = host[:i]
	}
	w.Header().Set("Strict-Transport-Security", "max-age=31536000")
	http.Redirect(w, r, "https://"+host+r.URL.RequestURI(), http.StatusPermanentRedirect)
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "panel: load config: %v\n", err)
		os.Exit(1)
	}

	log, err := logger.New(cfg.LogLevel, cfg.LogFormat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "panel: init logger: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = log.Sync() }()

	gdb, err := db.Open(cfg.MySQLDSN, log)
	if err != nil {
		log.Fatal("panel: open database", zap.Error(err))
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		log.Fatal("panel: get sql db", zap.Error(err))
	}

	secret := []byte(cfg.JWTSecret)
	secretSource := "env"
	if len(secret) == 0 {
		secret = auth.DeriveSecret(cfg.MySQLDSN + "|jwt")
		secretSource = "derived-from-dsn"
		log.Warn("panel: PANEL_JWT_SECRET not set, derive from MYSQL_DSN")
	}
	tokens := auth.NewManager(secret, cfg.AccessTTL, cfg.RefreshTTL)

	var svc *service.Service
	mqttCli, err := mqttcli.New(mqttcli.Config{
		Broker:   cfg.MQTTBroker,
		CAFile:   cfg.MQTTCACert,
		SNI:      cfg.MQTTSNI,
		User:     cfg.MQTTUser,
		Pass:     cfg.MQTTPass,
		ClientID: "panel-" + uuid.NewString()[:8],
	}, log, func(topic string, payload []byte) {
		if svc != nil {
			svc.HandleMessage(topic, payload)
		}
	}, []string{aoesignal.ReportFilter(), aoesignal.StateFilter()}, nil)
	if err != nil {
		log.Fatal("panel: init mqtt", zap.Error(err))
	}
	svc = service.New(gdb, mqttCli, tokens, log)

	// 节点令牌派生密钥：显式配置优先，缺省由 MYSQL_DSN 派生，保证重启后旧令牌不失效。
	nodeSecret := []byte(cfg.NodeSecret)
	if len(nodeSecret) == 0 {
		nodeSecret = auth.DeriveSecret(cfg.MySQLDSN + "|node")
		log.Warn("panel: PANEL_NODE_SECRET not set, derive from MYSQL_DSN")
	}
	svc.SetNodeSecret(nodeSecret)

	bootstrapCtx, bootstrapCancel := context.WithTimeout(context.Background(), 15*time.Second)
	if err := svc.Bootstrap(bootstrapCtx, cfg.AdminUser, cfg.AdminPass, cfg.AgentUser, cfg.AgentPass); err != nil {
		bootstrapCancel()
		log.Fatal("panel: bootstrap rbac", zap.Error(err))
	}
	bootstrapCancel()

	if err := mqttCli.Connect(); err != nil {
		// 面板不因控制面暂不可用而拒绝启动，paho 会在后台持续重连。
		log.Warn("panel: mqtt initial connect failed", zap.Error(err))
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "panel",
			"version": version.Version,
		})
	})
	api.Register(router, svc, api.Options{
		Tokens:           tokens,
		HTTPAddr:         cfg.HTTPAddr,
		MQTTBroker:       cfg.MQTTBroker,
		Version:          version.Version,
		JWTSecretSource:  secretSource,
		AgentDownloadURL: cfg.AgentDownloadURL,
		AgentUser:        cfg.AgentUser,
		AgentPass:        cfg.AgentPass,
		MQTTUser:         cfg.MQTTUser,
		MQTTPass:         cfg.MQTTPass,
		PublicURL:        cfg.PublicURL,
		AgentBroker:      cfg.AgentBroker,
	})

	// Agent 安装物料由主控面板直接提供，安装脚本只从云存储取 install.sh 自身。
	if cfg.AgentDistDir != "" {
		if err := serveDownloads(router, cfg.AgentDistDir); err != nil {
			log.Fatal("panel: serve agent downloads", zap.Error(err))
		}
		log.Info("panel: serve agent downloads", zap.String("dir", cfg.AgentDistDir))
	}

	// 配置了前端产物目录时，面板同时托管页面；API 与静态资源同域，省去额外反代。
	if cfg.WebDir != "" {
		if err := serveWeb(router, cfg.WebDir); err != nil {
			log.Fatal("panel: serve web", zap.Error(err))
		}
		log.Info("panel: serve web", zap.String("dir", cfg.WebDir))
	}

	tlsOn := cfg.TLSCert != "" && cfg.TLSKey != ""
	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}
	var redirectSrv *http.Server
	if tlsOn {
		// 开启 HTTPS 后 HTTP 端口只做跳转，之前分享出去的 http 链接仍然可用。
		srv.Addr = cfg.HTTPSAddr
		redirectSrv = &http.Server{
			Addr:              cfg.HTTPAddr,
			Handler:           http.HandlerFunc(redirectHTTPS),
			ReadHeaderTimeout: 5 * time.Second,
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	// 心跳超时的部署实例需要定期转为离线，并让入口规则剔除不可达地址。
	go svc.RunInstanceSweeper(ctx)

	errCh := make(chan error, 1)
	go func() {
		log.Info("panel listening", zap.String("addr", srv.Addr), zap.Bool("tls", tlsOn), zap.String("version", version.Version))
		if tlsOn {
			errCh <- srv.ListenAndServeTLS(cfg.TLSCert, cfg.TLSKey)
			return
		}
		errCh <- srv.ListenAndServe()
	}()
	if redirectSrv != nil {
		go func() {
			if err := redirectSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Warn("panel http redirect stopped", zap.Error(err))
			}
		}()
	}

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("panel server error", zap.Error(err))
		}
	case <-ctx.Done():
		log.Info("panel shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("panel shutdown error", zap.Error(err))
	}
	if redirectSrv != nil {
		if err := redirectSrv.Shutdown(shutdownCtx); err != nil {
			log.Warn("panel http redirect shutdown error", zap.Error(err))
		}
	}
	mqttCli.Disconnect()
	if err := sqlDB.Close(); err != nil {
		log.Warn("panel: close database", zap.Error(err))
	}
}
