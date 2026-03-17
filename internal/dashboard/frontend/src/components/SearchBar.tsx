import { useState, useCallback, useEffect, useRef } from 'react';
import { fetchSearch } from '../api/client';
import type { SearchResponse } from '../types';

interface SearchBarProps {
  host: string;
}

type SearchType = 'all' | 'memory' | 'note';

function truncate(s: string, n: number): string {
  return s.length > n ? s.slice(0, n) + '…' : s;
}

export function SearchBar({ host }: SearchBarProps) {
  const [query, setQuery] = useState('');
  const [searchType, setSearchType] = useState<SearchType>('all');
  const [results, setResults] = useState<SearchResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const doSearch = useCallback(async (q: string, type: SearchType, h: string) => {
    if (!q.trim()) {
      setResults(null);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const data = await fetchSearch({ q, type, host: h || undefined, limit: 10 });
      setResults(data);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Search failed');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      doSearch(query, searchType, host);
    }, 350);
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [query, searchType, host, doSearch]);

  const totalResults = results
    ? results.memories.length + results.notes.length
    : 0;

  return (
    <div>
      <div style={{ display: 'flex', gap: 8 }}>
        <input
          type="text"
          placeholder="Search memories and notes…"
          value={query}
          onChange={e => setQuery(e.target.value)}
          style={{
            flex: 1,
            padding: '8px 12px',
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 'var(--radius)',
            color: 'var(--text)',
          }}
        />
        <select
          value={searchType}
          onChange={e => setSearchType(e.target.value as SearchType)}
          style={{
            padding: '8px 12px',
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 'var(--radius)',
            color: 'var(--text)',
          }}
        >
          <option value="all">All</option>
          <option value="memory">Memories</option>
          <option value="note">Notes</option>
        </select>
      </div>

      {loading && (
        <div style={{ padding: '12px 0', color: 'var(--text-muted)', fontSize: 13 }}>
          Searching…
        </div>
      )}
      {error && (
        <div style={{ padding: '12px 0', color: 'var(--error)', fontSize: 13 }}>
          {error}
        </div>
      )}
      {results && !loading && (
        <div style={{ marginTop: 12 }}>
          {totalResults === 0 ? (
            <div style={{ color: 'var(--text-dim)', fontSize: 13 }}>No results found.</div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              {results.memories.map(m => (
                <div key={`mem-${m.id}`} style={{
                  padding: '10px 14px',
                  background: 'var(--bg-card)',
                  border: '1px solid var(--border)',
                  borderRadius: 'var(--radius)',
                }}>
                  <div style={{ display: 'flex', gap: 8, alignItems: 'center', marginBottom: 4 }}>
                    <span style={{
                      fontSize: 10, fontWeight: 700, padding: '1px 6px',
                      borderRadius: 3, background: '#6366f122', color: '#6366f1',
                      textTransform: 'uppercase',
                    }}>memory</span>
                    <span style={{ fontFamily: 'var(--font-mono)', fontSize: 13 }}>{m.key}</span>
                    <span style={{ marginLeft: 'auto', fontSize: 11, color: 'var(--text-dim)' }}>
                      {m.category} · {m.host}
                    </span>
                  </div>
                  <div style={{ color: 'var(--text-muted)', fontSize: 13 }}>
                    {truncate(m.value, 100)}
                  </div>
                </div>
              ))}
              {results.notes.map(n => (
                <div key={`note-${n.id}`} style={{
                  padding: '10px 14px',
                  background: 'var(--bg-card)',
                  border: '1px solid var(--border)',
                  borderRadius: 'var(--radius)',
                }}>
                  <div style={{ display: 'flex', gap: 8, alignItems: 'center', marginBottom: 4 }}>
                    <span style={{
                      fontSize: 10, fontWeight: 700, padding: '1px 6px',
                      borderRadius: 3, background: '#22c55e22', color: '#22c55e',
                      textTransform: 'uppercase',
                    }}>note</span>
                    <span style={{ fontWeight: 600 }}>{n.title}</span>
                    <span style={{ marginLeft: 'auto', fontSize: 11, color: 'var(--text-dim)' }}>
                      {n.project} · {n.host}
                    </span>
                  </div>
                  <div style={{ color: 'var(--text-muted)', fontSize: 13 }}>
                    {truncate(n.content, 100)}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
