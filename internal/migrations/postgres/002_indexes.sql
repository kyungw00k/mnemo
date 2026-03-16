CREATE INDEX IF NOT EXISTS idx_memories_host ON memories(host);
CREATE INDEX IF NOT EXISTS idx_notes_host ON notes(host);

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'vector') THEN
        EXECUTE 'CREATE INDEX IF NOT EXISTS idx_memories_embedding ON memories USING hnsw (embedding vector_cosine_ops)';
        EXECUTE 'CREATE INDEX IF NOT EXISTS idx_notes_embedding ON notes USING hnsw (embedding vector_cosine_ops)';
    END IF;
END $$;
