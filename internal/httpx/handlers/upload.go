package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/RENCHILIU/gallerio/internal/httpx/middleware"
	"github.com/RENCHILIU/gallerio/internal/store"
)

type UploadHandler struct {
	Store       *store.Store
	DataDir     string
	UploadMaxMB int // 每个文件的最大大小（MB），来自 config.UploadMaxMB
}

func NewUploadHandler(s *store.Store, dataDir string, uploadMaxMB int) *UploadHandler {
	return &UploadHandler{Store: s, DataDir: dataDir, UploadMaxMB: uploadMaxMB}
}

type uploadSaved struct {
	ID  int64  `json:"id"`
	URL string `json:"url"`
}

type uploadError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId,omitempty"`
}

// POST /api/upload
// 表单字段名：files（可多选）
// 返回：{ saved: [{id, url}], count: N }
func (h *UploadHandler) Upload(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.RequestIDFromContext(r.Context())

	// 解析 multipart（内存上限只是“阈值”，超出会落临时文件；真正的大小限制在我们自定义拷贝里做）
	const mem = 10 << 20 // 10MB
	if err := r.ParseMultipartForm(mem); err != nil {
		writeJSON(w, http.StatusBadRequest, uploadError{"BAD_PAYLOAD", "invalid multipart form", reqID})
		return
	}
	form := r.MultipartForm
	files := form.File["files"]
	if len(files) == 0 {
		writeJSON(w, http.StatusBadRequest, uploadError{"NO_FILES", "no files provided (use field name 'files')", reqID})
		return
	}

	var saved []uploadSaved
	for _, fh := range files {
		urlPath, diskPath, size, mime, err := h.saveOneFile(fh)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, uploadError{"UPLOAD_FAILED", err.Error(), reqID})
			return
		}

		// 写 DB（失败时回滚文件）
		p := store.Photo{
			FileName:     fh.Filename,
			MimeType:     strPtrOrNil(zeroIsEmpty(mime)),
			SizeBytes:    size,
			PathOriginal: urlPath,
			UploadedAt:   time.Now().UTC(),
		}
		id, err := h.Store.InsertPhoto(r.Context(), &p)
		if err != nil {
			log.Printf("upload insert error: %v (file_name=%q path=%q)", err, p.FileName, p.PathOriginal)
			_ = removeFileQuiet(diskPath)
			writeJSON(w, http.StatusInternalServerError, uploadError{"INTERNAL", "database error", reqID})
			return
		}
		saved = append(saved, uploadSaved{ID: id, URL: urlPath})
	}

	writeJSON(w, http.StatusOK, struct {
		Saved []uploadSaved `json:"saved"`
		Count int           `json:"count"`
	}{Saved: saved, Count: len(saved)})
}

// ---- helpers ----

func (h *UploadHandler) saveOneFile(fh *multipart.FileHeader) (urlPath, diskPath string, size int64, mime string, err error) {
	maxBytes := int64(h.UploadMaxMB) * 1024 * 1024
	if maxBytes <= 0 {
		return "", "", 0, "", errors.New("upload max size misconfigured")
	}

	// 目录：{DataDir}/photos/original/YYYY/MM
	now := time.Now().UTC()
	sub := filepath.Join("photos", "original", now.Format("2006"), now.Format("01"))
	targetDir := filepath.Join(h.DataDir, sub)
	if err = mkdirAll(targetDir); err != nil {
		return "", "", 0, "", err
	}

	base := sanitize(fh.Filename)
	if base == "" {
		base = "unnamed"
	}
	// 避免冲突：加 12 字节随机 + 时间戳前缀
	base = tsRand() + "_" + base
	diskPath = filepath.Join(targetDir, base)
	urlPath = filepath.ToSlash(filepath.Join("/media", sub, base))

	src, err := fh.Open()
	if err != nil {
		return "", "", 0, "", err
	}
	defer src.Close()

	dst, err := createFile(diskPath)
	if err != nil {
		return "", "", 0, "", err
	}
	defer func() {
		_ = dst.Close()
		if err != nil {
			_ = removeFileQuiet(diskPath) // 失败回滚
		}
	}()

	// 先读 512 字节嗅探类型
	head := make([]byte, 512)
	nHead, _ := io.ReadFull(src, head)
	if nHead > 0 {
		if _, e := dst.Write(head[:nHead]); e != nil {
			return "", "", 0, "", e
		}
	}
	size = int64(nHead)
	mime = http.DetectContentType(head[:nHead])
	if !isAllowedImage(mime, fh.Filename) {
		return "", "", 0, "", errors.New("file type not allowed")
	}
	if size > maxBytes {
		return "", "", 0, "", errors.New("file exceeds max size")
	}

	// 继续拷贝，严格限制最大大小
	buf := make([]byte, 32*1024)
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			remain := maxBytes - size
			if remain <= 0 {
				return "", "", 0, "", errors.New("file exceeds max size")
			}
			if int64(nr) > remain {
				// 只写到上限，返回错误
				if _, e := dst.Write(buf[:remain]); e != nil {
					return "", "", 0, "", e
				}
				size += remain
				return "", "", 0, "", errors.New("file exceeds max size")
			}
			if _, e := dst.Write(buf[:nr]); e != nil {
				return "", "", 0, "", e
			}
			size += int64(nr)
		}
		if er != nil {
			if er == io.EOF {
				break
			}
			return "", "", 0, "", er
		}
	}
	return urlPath, diskPath, size, mime, nil
}

func isAllowedImage(mime, name string) bool {
	m := strings.ToLower(mime)
	if strings.HasPrefix(m, "image/jpeg") ||
		strings.HasPrefix(m, "image/png") ||
		strings.HasPrefix(m, "image/gif") ||
		strings.HasPrefix(m, "image/webp") {
		return true
	}
	// 兜底按扩展名
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return true
	default:
		return false
	}
}

func sanitize(name string) string {
	name = filepath.Base(name)
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "..", "_")
	return name
}

func tsRand() string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return time.Now().UTC().Format("20060102T150405") + "_" + hex.EncodeToString(b[:])
}

func mkdirAll(p string) error               { return ensureDir(p) } // wrapper，便于后续替换
func ensureDir(p string) error              { return os.MkdirAll(p, 0o755) }
func createFile(p string) (*os.File, error) { return os.Create(p) }
func removeFileQuiet(p string) error        { return os.Remove(p) }

func zeroIsEmpty(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}

func strPtrOrNil(s *string) *string { return s }
