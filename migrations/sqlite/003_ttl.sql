ALTER TABLE memories ADD COLUMN expires_at DATETIME;
ALTER TABLE notes ADD COLUMN expires_at DATETIME;
CREATE INDEX IF NOT EXISTS idx_memories_expires_at ON memories(expires_at);
CREATE INDEX IF NOT EXISTS idx_notes_expires_at ON notes(expires_at);
