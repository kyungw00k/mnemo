CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS memories (
    id BIGSERIAL PRIMARY KEY,
    host VARCHAR(255) NOT NULL,
    category VARCHAR(100) NOT NULL,
    memory_key VARCHAR(255) NOT NULL,
    memory_value TEXT NOT NULL,
    metadata TEXT,
    embedding vector($DIMENSIONS),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    del_yn CHAR(1) DEFAULT 'N',
    UNIQUE(host, category, memory_key)
);

CREATE TABLE IF NOT EXISTS notes (
    id BIGSERIAL PRIMARY KEY,
    host VARCHAR(255) NOT NULL,
    project VARCHAR(100),
    title VARCHAR(500) NOT NULL,
    content TEXT NOT NULL,
    tags TEXT,
    embedding vector($DIMENSIONS),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    del_yn CHAR(1) DEFAULT 'N'
);
