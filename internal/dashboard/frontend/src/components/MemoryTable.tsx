import type { MemoryItem } from '../types';

interface MemoryTableProps {
  items: MemoryItem[];
  loading: boolean;
  error: string | null;
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

const CATEGORY_COLORS: Record<string, string> = {
  decision:   '#6366f1',
  convention: '#22c55e',
  config:     '#f59e0b',
  bug:        '#ef4444',
  preference: '#ec4899',
};

function categoryColor(cat: string): string {
  return CATEGORY_COLORS[cat.toLowerCase()] ?? '#64748b';
}

export function MemoryTable({ items, loading, error }: MemoryTableProps) {
  if (loading) {
    return (
      <div style={{ padding: 24, color: 'var(--text-muted)', textAlign: 'center' }}>
        Loading memories…
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
        No memories found.
      </div>
    );
  }

  return (
    <div style={{ overflowX: 'auto' }}>
      <table style={{ width: '100%', borderCollapse: 'collapse' }}>
        <thead>
          <tr style={{ borderBottom: '1px solid var(--border)' }}>
            {(['Category', 'Key', 'Value', 'Host', 'Updated'] as const).map(h => (
              <th key={h} style={{
                padding: '10px 12px',
                textAlign: 'left',
                color: 'var(--text-muted)',
                fontSize: 12,
                fontWeight: 500,
                textTransform: 'uppercase',
                letterSpacing: '0.05em',
                whiteSpace: 'nowrap',
              }}>
                {h}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {items.map(m => (
            <tr key={m.id} style={{
              borderBottom: '1px solid var(--border)',
              transition: 'background 0.1s',
            }}
              onMouseEnter={e => (e.currentTarget.style.background = 'var(--bg-hover)')}
              onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
            >
              <td style={{ padding: '10px 12px' }}>
                <span style={{
                  display: 'inline-block',
                  padding: '2px 8px',
                  borderRadius: 4,
                  fontSize: 11,
                  fontWeight: 600,
                  background: categoryColor(m.category) + '22',
                  color: categoryColor(m.category),
                  border: `1px solid ${categoryColor(m.category)}44`,
                }}>
                  {m.category}
                </span>
              </td>
              <td style={{ padding: '10px 12px', fontFamily: 'var(--font-mono)', fontSize: 13 }}>
                {truncate(m.key, 40)}
              </td>
              <td style={{ padding: '10px 12px', color: 'var(--text-muted)' }}>
                {truncate(m.value, 60)}
              </td>
              <td style={{ padding: '10px 12px', color: 'var(--text-dim)', fontSize: 12 }}>
                {m.host}
              </td>
              <td style={{ padding: '10px 12px', color: 'var(--text-dim)', fontSize: 12, whiteSpace: 'nowrap' }}>
                {formatDate(m.updated_at)}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
