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
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/company/ems-devices/internal/collectors"
	"github.com/company/ems-devices/internal/config"
)

func main() {
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

	// 启动采集主循环。
	go service.Run(ctx)

	// 暴露 Prometheus 指标。
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	server := &http.Server{
		Addr:    cfg.Prometheus.ListenAddr(),
		Handler: mux,
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
