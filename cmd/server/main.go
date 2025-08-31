package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/RENCHILIU/gallerio/internal/config"
	"github.com/RENCHILIU/gallerio/internal/httpx/middleware"
)

func init() {
	// 把 slog 设置为 Debug handler，输出更多字段
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)
}

func main() {
	config.LoadDotEnv(".env")
	cfg := config.Load()
	slog.Info("starting gallerio", "addr", cfg.AppAddr, "dataDir", cfg.DataDir)

	// 确保数据目录存在
	if err := os.MkdirAll(filepath.Join(cfg.DataDir, "photos", "original"), 0o755); err != nil {
		slog.Error("mkdir data", "err", err)
		os.Exit(1)
	}

	// 连接 MySQL
	db, err := sql.Open("mysql", cfg.MysqlDSN)
	if err != nil {
		slog.Error("sql.Open", "err", err)
		os.Exit(1)
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(20)
	db.SetConnMaxLifetime(time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		slog.Error("db ping failed", "err", err)
		os.Exit(1)
	}
	slog.Info("db connected")

	// 路由
	mux := http.NewServeMux()

	// 首页
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/templates/index.html")
	})

	// Swagger 文档
	mux.Handle("/docs/", http.StripPrefix("/docs/", http.FileServer(http.Dir("web/docs"))))

	// 静态资源（如果你放 JS/CSS）
	mux.Handle("/web/", http.StripPrefix("/web/", http.FileServer(http.Dir("web/static"))))

	// 原图（禁止目录列出）
	mux.Handle("/media/original/",
		http.StripPrefix("/media/original/",
			middleware.NoDirFileServer(filepath.Join(cfg.DataDir, "photos", "original")),
		),
	)

	// 健康检查
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"ok":false}`))
			return
		}
		w.Write([]byte(`{"ok":true}`))
	})

	// 包裹中间件
	handler := middleware.Recover(middleware.Logging(mux))

	srv := &http.Server{
		Addr:              cfg.AppAddr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// 优雅关闭
	go func() {
		slog.Info("http listening", "addr", cfg.AppAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("ListenAndServe", "err", err)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("shutting down...")
	shutCtx, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	_ = srv.Shutdown(shutCtx)
	_ = db.Close()
	slog.Info("bye")
}
