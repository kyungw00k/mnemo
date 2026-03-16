CREATE VIRTUAL TABLE IF NOT EXISTS vec_memories USING vec0(
    memory_id INTEGER PRIMARY KEY,
    embedding float[$DIMENSIONS]
);

CREATE VIRTUAL TABLE IF NOT EXISTS vec_notes USING vec0(
    note_id INTEGER PRIMARY KEY,
    embedding float[$DIMENSIONS]
);
