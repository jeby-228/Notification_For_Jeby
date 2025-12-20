-- 數據庫遷移腳本
-- 為 members 表添加 password_hash 字段以支持用戶認證

-- 如果表不存在，創建完整的表結構
CREATE TABLE IF NOT EXISTS members (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255),
    isDeleted BOOLEAN DEFAULT FALSE,
    deletedAt TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 如果表已存在但缺少 password_hash 字段，添加該字段
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'members'
        AND column_name = 'password_hash'
    ) THEN  
        ALTER TABLE members ADD COLUMN password_hash VARCHAR(255);
        ALTER TABLE members ADD COLUMN isDeleted BOOLEAN DEFAULT FALSE;
        ALTER TABLE members ADD COLUMN deletedAt TIMESTAMP;
    END IF;
END $$;
