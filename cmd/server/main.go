package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"runtime"

	"github.com/RENCHILIU/gallerio/internal/config"
	"github.com/RENCHILIU/gallerio/internal/httpx/handlers"
	"github.com/RENCHILIU/gallerio/internal/httpx/middleware"
	"github.com/RENCHILIU/gallerio/internal/store"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// 1) 读取 .env（开发环境用；线上容器里可不需要）
	config.LoadDotEnv(".env.local")
	config.LoadDotEnv(".env")

	// 2) 装载配置（从环境变量读取，带默认值）
	cfg := config.Load()

	// 3) 初始化数据库（确保 DSN 带 parseTime=true）
	db, err := sql.Open("mysql", cfg.MysqlDSN)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("ping db: %v", err)
	}
	defer db.Close()

	// 4) 依赖注入：store & handlers
	st := store.New(db)

	const maxPageSize = 500
	photos := handlers.NewPhotosHandler(st, cfg.PageSize, maxPageSize)
	upload := handlers.NewUploadHandler(st, cfg.DataDir, cfg.UploadMaxMB)

	// 5) 解析模板（与启动目录无关）
	tpl := mustLoadTemplates()
	web := handlers.NewWebHandler(tpl, cfg.PageSize, cfg.UploadMaxMB)

	// 6) 路由
	mux := http.NewServeMux()

	// 健康检查
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	// API
	mux.HandleFunc("/api/photos", photos.List)
	mux.HandleFunc("/api/upload", upload.Upload)

	// 页面
	mux.HandleFunc("/", web.Index)
	mux.HandleFunc("/slideshow", web.Slideshow)

	// 静态：/media -> DATA_DIR（例如 ./data）
	fs := http.FileServer(http.Dir(cfg.DataDir))
	mux.Handle("/media/", http.StripPrefix("/media/", fs))

	// 7) 中间件：RequestID -> 访问日志
	var handler http.Handler = mux
	handler = middleware.RequestID(handler)
	handler = middleware.AccessLog(handler)

	log.Printf("listening on %s | DATA_DIR=%s", cfg.AppAddr, cfg.DataDir)
	log.Fatal(http.ListenAndServe(cfg.AppAddr, handler))
}

// mustLoadTemplates 通过当前源文件路径反推项目根，解析模板文件。
// 无论你从仓库根还是 cmd/server 目录启动，都能找到模板。
func mustLoadTemplates() *template.Template {
	// file 是当前 main.go 的绝对路径：.../cmd/server/main.go
	_, file, _, _ := runtime.Caller(0)
	// 项目根：从 cmd/server 回退两级到仓库根
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	tplDir := filepath.Join(repoRoot, "web", "templates")

	// 解析 index.tmpl + slideshow.tmpl
	return template.Must(template.ParseFiles(
		filepath.Join(tplDir, "index.tmpl"),
		filepath.Join(tplDir, "slideshow.tmpl"),
	))
}
