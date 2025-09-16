package store

import (
	"context"
	"database/sql"
	"time"
)

// 领域模型（与表字段一一对应）
type Photo struct {
	ID           int64
	FileName     string
	MimeType     *string // 允许为 NULL
	SizeBytes    int64
	PathOriginal string
	UploadedAt   time.Time // 需在 DSN 加 parseTime=true
}

type ListResult struct {
	Items []Photo
	Total int
}

type Store struct {
	DB *sql.DB
}

// New 仅保存连接；DB 生命周期由上层负责（main 中创建/关闭）
func New(db *sql.DB) *Store { return &Store{DB: db} }

// ListPhotos 列表 + 总数（LIMIT/OFFSET）
// 说明：limit/offset 的校验放在 handler 层；这里假设入参已合法。
func (s *Store) ListPhotos(ctx context.Context, limit, offset int) (ListResult, error) {
	// 防止慢查询拖挂：给数据库操作统一加超时（也可让上层传入带超时的 ctx）
	const dbTimeout = 2 * time.Second
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	const listSQL = `
SELECT id, file_name, mime_type, size_bytes, path_original, uploaded_at
FROM photos
ORDER BY uploaded_at DESC, id DESC
LIMIT ? OFFSET ?;`

	rows, err := s.DB.QueryContext(ctx, listSQL, limit, offset)
	if err != nil {
		return ListResult{}, err
	}
	defer rows.Close()

	items := make([]Photo, 0, limit)
	for rows.Next() {
		var p Photo
		var mime sql.NullString
		if err := rows.Scan(&p.ID, &p.FileName, &mime, &p.SizeBytes, &p.PathOriginal, &p.UploadedAt); err != nil {
			return ListResult{}, err
		}
		if mime.Valid {
			p.MimeType = &mime.String
		}
		items = append(items, p)
	}
	if err := rows.Err(); err != nil {
		return ListResult{}, err
	}

	// 统计总数（给前端 hasMore/滚动判断）
	const countSQL = `SELECT COUNT(*) FROM photos;`
	var total int
	if err := s.DB.QueryRowContext(ctx, countSQL).Scan(&total); err != nil {
		return ListResult{}, err
	}

	return ListResult{Items: items, Total: total}, nil
}
