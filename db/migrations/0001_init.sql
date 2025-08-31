CREATE DATABASE slideshow
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_0900_ai_ci;
  
  
  -- photos 表（MVP）
CREATE TABLE IF NOT EXISTS photos (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  file_name VARCHAR(255) NOT NULL,
  mime_type VARCHAR(100) NOT NULL,
  size_bytes BIGINT NOT NULL,
  path_original VARCHAR(512) NOT NULL,
  width INT NULL,
  height INT NULL,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  uploaded_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  KEY idx_created_at (created_at DESC, id DESC),
  KEY idx_file_name (file_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

