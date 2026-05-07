package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/company/ems-devices/internal/alerts"
	"github.com/company/ems-devices/internal/api"
	"github.com/company/ems-devices/internal/collectors"
	"github.com/company/ems-devices/internal/config"
	"github.com/company/ems-devices/internal/notifier"
	"github.com/company/ems-devices/internal/routes"
)

func main() {
	// 清除代理设置（避免数据库连接被代理拦截）
	os.Unsetenv("http_proxy")
	os.Unsetenv("https_proxy")
	os.Unsetenv("HTTP_PROXY")
	os.Unsetenv("HTTPS_PROXY")
	os.Unsetenv("all_proxy")
	os.Unsetenv("ALL_PROXY")

	if err := loadEnv(); err != nil {
		log.Fatalf("加载环境变量失败: %v", err)
	}

	var configPath string
	flag.StringVar(&configPath, "config", "configs/config.yml", "配置文件路径")
	flag.Parse()

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("载入配置失败: %v", err)
	}

	service, err := collectors.NewService(cfg)
	if err != nil {
		log.Fatalf("初始化采集服务失败: %v", err)
	}
	defer service.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 初始化告警组件
	alertStoragePath := "configs/alerts.json"
	alertStorage := alerts.NewStorage(alertStoragePath)
	if err := alertStorage.Load(); err != nil {
		log.Printf("警告: 加载告警规则失败，将创建新的: %v", err)
	}

	alertHistory := alerts.NewHistory(1000) // 保留最近 1000 条历史
	metricStore := alerts.NewMetricValueStore(48 * time.Hour) // 保留 48 小时数据

	// 初始化告警评估器
	var alertmanager *alerts.AlertmanagerClient

	// 只有在内置通知服务未启用时，才初始化外部 Alertmanager
	if cfg.Notifier.Enabled {
		log.Printf("[NOTIFIER] 使用内置通知服务，不使用外部 Alertmanager")
		alertmanager = nil
	} else {
		// 初始化 Alertmanager 客户端（从配置或环境变量读取）
		alertmanagerURL := cfg.Alertmanager.URL
		if alertmanagerURL == "" {
			// 兼容环境变量
			alertmanagerURL = os.Getenv("ALERTMANAGER_URL")
			if alertmanagerURL == "" {
				alertmanagerURL = "http://localhost:9093"
			}
		}
		alertmanager = alerts.NewAlertmanagerClient(alertmanagerURL)
		log.Printf("Alertmanager 地址: %s", alertmanagerURL)
	}

	alertEvaluator := alerts.NewEvaluator(alertStorage, alertHistory, alertmanager, metricStore, service)
	service.SetAlertEvaluator(alertEvaluator)

	// 初始化内置告警通知服务（如果配置启用）
	if cfg.Notifier.Enabled {
		log.Printf("[NOTIFIER] 初始化内置告警通知服务...")
		notifierCfg := notifier.FromConfig(&cfg.Notifier)
		notifierMgr := notifier.NewLegacyManager(notifierCfg)
		alertEvaluator.SetNotifier(notifierMgr)
		log.Printf("[NOTIFIER] 内置告警通知服务已启用，仅使用内置通知发送告警")
	}

	// 启动采集主循环
	go service.Run(ctx)

	// 启动告警定时评估循环
	go service.RunScheduledEvaluation(ctx)

	// 暴露 Prometheus 指标和 API
	apiServer := api.NewServer(configPath, service)

	// 设置告警 API handler
	alertHandler := alerts.NewHandler(alertStorage, alertHistory, alertEvaluator)
	apiServer.SetAlertHandler(alertHandler)

	// 初始化路由管理器
	routeStorage := routes.NewRouteStorage("data/routes.json")
	if err := routeStorage.Load(); err != nil {
		log.Printf("[ROUTE] 加载路由配置失败: %v", err)
	}
	routeMgr := routes.NewManager(routeStorage)
	routeHandler := routes.NewHandler(routeStorage, routeMgr)
	apiServer.SetRouteHandler(routeHandler)
	log.Printf("[ROUTE] 路由管理器已初始化")

	server := &http.Server{
		Addr:    cfg.Prometheus.ListenAddr(),
		Handler: apiServer,
	}

	go func() {
		log.Printf("Prometheus 指标监听地址: %s", cfg.Prometheus.ListenAddr())
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP 服务异常退出: %v", err)
		}
	}()

	// 捕获系统信号，实现优雅退出。
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Println("收到终止信号，准备退出...")

	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("关闭 HTTP 服务失败: %v", err)
	}
	log.Println("采集器已退出。")
}

func loadEnv() error {
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(".env"); err != nil {
			return err
		}
	}
	return nil
}
