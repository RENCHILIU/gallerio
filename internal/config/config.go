package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// very small .env loader (no deps)
func LoadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if i := strings.IndexByte(line, '='); i > 0 {
			k := strings.TrimSpace(line[:i])
			v := strings.TrimSpace(line[i+1:])
			// strip optional quotes
			v = strings.Trim(v, `"'`)
			if os.Getenv(k) == "" {
				os.Setenv(k, v)
			}
		}
	}
}

type Config struct {
	AppAddr     string
	MysqlDSN    string
	DataDir     string
	UploadMaxMB int
	PageSize    int
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
func getenvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func Load() Config {
	return Config{
		AppAddr:     getenv("APP_ADDR", ":8080"),
		MysqlDSN:    getenv("MYSQL_DSN", "slideshow:changeme@tcp(127.0.0.1:3306)/slideshow?parseTime=true&charset=utf8mb4"),
		DataDir:     getenv("DATA_DIR", "./data"),
		UploadMaxMB: getenvInt("UPLOAD_MAX_MB", 30),
		PageSize:    getenvInt("PAGE_SIZE", 50),
	}
}
