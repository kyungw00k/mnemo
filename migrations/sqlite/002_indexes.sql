CREATE INDEX IF NOT EXISTS idx_memories_host ON memories(host);
CREATE INDEX IF NOT EXISTS idx_notes_host ON notes(host);
CREATE INDEX IF NOT EXISTS idx_memories_category ON memories(host, category);
