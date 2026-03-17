import type { NoteItem } from '../types';

interface NoteListProps {
  items: NoteItem[];
  loading: boolean;
  error: string | null;
  onNoteClick?: (id: number) => void;
}

function formatDate(iso: string): string {
  try {
    return new Date(iso).toLocaleString(undefined, {
      month: 'short', day: 'numeric',
      hour: '2-digit', minute: '2-digit',
    });
  } catch {
    return iso;
  }
}

function truncate(s: string, n: number): string {
  return s.length > n ? s.slice(0, n) + '…' : s;
}

export function NoteList({ items, loading, error, onNoteClick }: NoteListProps) {
  if (loading) {
    return (
      <div style={{ padding: 24, color: 'var(--text-muted)', textAlign: 'center' }}>
        Loading notes…
      </div>
    );
  }
  if (error) {
    return (
      <div style={{ padding: 24, color: 'var(--error)', textAlign: 'center' }}>
        {error}
      </div>
    );
  }
  if (items.length === 0) {
    return (
      <div style={{ padding: 24, color: 'var(--text-dim)', textAlign: 'center' }}>
        No notes found.
      </div>
    );
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
      {items.map(note => (
        <div
          key={note.id}
          style={{
            padding: '14px 16px',
            borderBottom: '1px solid var(--border)',
            cursor: onNoteClick ? 'pointer' : 'default',
            transition: 'background 0.1s',
          }}
          onClick={() => onNoteClick?.(note.id)}
          onMouseEnter={e => (e.currentTarget.style.background = 'var(--bg-hover)')}
          onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
        >
          <div style={{ display: 'flex', alignItems: 'baseline', gap: 8, marginBottom: 4 }}>
            <span style={{ fontWeight: 600, color: 'var(--text)' }}>
              {note.title}
            </span>
            {note.project && (
              <span style={{
                fontSize: 11,
                padding: '1px 6px',
                borderRadius: 4,
                background: 'var(--bg)',
                border: '1px solid var(--border)',
                color: 'var(--text-muted)',
              }}>
                {note.project}
              </span>
            )}
          </div>
          <div style={{ color: 'var(--text-muted)', fontSize: 13, marginBottom: 6 }}>
            {truncate(note.content, 120)}
          </div>
          <div style={{ display: 'flex', gap: 12, fontSize: 11, color: 'var(--text-dim)' }}>
            <span>{note.host}</span>
            <span>{formatDate(note.updated_at)}</span>
            {note.tags && (
              <span style={{ color: 'var(--accent)' }}>
                {note.tags.split(',').map(t => `#${t.trim()}`).join(' ')}
              </span>
            )}
          </div>
        </div>
      ))}
    </div>
  );
}
