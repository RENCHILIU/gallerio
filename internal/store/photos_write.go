package store

import (
	"context"
	"database/sql"
	"time"
)

// InsertPhoto 写入一条照片记录，返回自增 ID。
// 说明：CreatedAt 先置 NULL（后续要做 EXIF 再补）；UploadedAt 必填。
func (s *Store) InsertPhoto(ctx context.Context, p *Photo) (int64, error) {
	const insertSQL = `
INSERT INTO photos (file_name, mime_type, size_bytes, path_original, created_at, uploaded_at)
VALUES (?, ?, ?, ?, ?, ?)`

	var createdAt sql.NullTime // NULL
	res, err := s.DB.ExecContext(ctx, insertSQL,
		p.FileName, p.MimeType, p.SizeBytes, p.PathOriginal, createdAt, p.UploadedAt,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// 小工具：现在时间（便于以后做可测试性注入）
func nowUTC() time.Time { return time.Now().UTC() }
