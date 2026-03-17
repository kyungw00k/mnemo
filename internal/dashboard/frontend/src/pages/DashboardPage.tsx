import { useState, useEffect, useCallback } from 'react';
import { fetchStats, fetchMemories, fetchNotes } from '../api/client';
import type { StatsData, MemoryItem, NoteItem } from '../types';
import { StatsCard } from '../components/StatsCard';
import { MemoryTable } from '../components/MemoryTable';
import { NoteList } from '../components/NoteList';
import { SearchBar } from '../components/SearchBar';

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section style={{
      background: 'var(--bg-card)',
      border: '1px solid var(--border)',
      borderRadius: 'var(--radius)',
      overflow: 'hidden',
    }}>
      <div style={{
        padding: '14px 20px',
        borderBottom: '1px solid var(--border)',
        fontWeight: 600,
        fontSize: 14,
        color: 'var(--text)',
      }}>
        {title}
      </div>
      {children}
    </section>
  );
}

export function DashboardPage() {
  const [stats, setStats] = useState<StatsData | null>(null);
  const [statsError, setStatsError] = useState<string | null>(null);

  const [selectedHost, setSelectedHost] = useState('');

  const [memories, setMemories] = useState<MemoryItem[]>([]);
  const [memoriesLoading, setMemoriesLoading] = useState(true);
  const [memoriesError, setMemoriesError] = useState<string | null>(null);

  const [notes, setNotes] = useState<NoteItem[]>([]);
  const [notesLoading, setNotesLoading] = useState(true);
  const [notesError, setNotesError] = useState<string | null>(null);

  // Fetch stats once
  useEffect(() => {
    fetchStats()
      .then(setStats)
      .catch(e => setStatsError(e instanceof Error ? e.message : 'Failed to load stats'));
  }, []);

  // Fetch memories when host changes
  const loadMemories = useCallback(async (host: string) => {
    setMemoriesLoading(true);
    setMemoriesError(null);
    try {
      const res = await fetchMemories({ host: host || undefined, limit: 50 });
      setMemories(res.items);
    } catch (e) {
      setMemoriesError(e instanceof Error ? e.message : 'Failed to load memories');
    } finally {
      setMemoriesLoading(false);
    }
  }, []);

  const loadNotes = useCallback(async (host: string) => {
    setNotesLoading(true);
    setNotesError(null);
    try {
      const res = await fetchNotes({ host: host || undefined, limit: 20 });
      setNotes(res.items);
    } catch (e) {
      setNotesError(e instanceof Error ? e.message : 'Failed to load notes');
    } finally {
      setNotesLoading(false);
    }
  }, []);

  useEffect(() => {
    loadMemories(selectedHost);
    loadNotes(selectedHost);
  }, [selectedHost, loadMemories, loadNotes]);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 24, padding: 24 }}>
      {/* Stats row */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 16 }}>
        <StatsCard
          label="Total Memories"
          value={stats ? stats.memories : '—'}
          sub={stats ? `${(stats.categories ?? []).length} categories` : undefined}
        />
        <StatsCard
          label="Total Notes"
          value={stats ? stats.notes : '—'}
          sub={stats ? `across ${(stats.hosts ?? []).length} hosts` : undefined}
        />
        <StatsCard
          label="Active Categories"
          value={stats ? (stats.categories ?? []).length : '—'}
          sub={stats ? (stats.categories ?? []).slice(0, 3).join(', ') + ((stats.categories ?? []).length > 3 ? '…' : '') : undefined}
        />
      </div>

      {statsError && (
        <div style={{ color: 'var(--error)', fontSize: 13 }}>
          Stats error: {statsError}
        </div>
      )}

      {/* Host filter */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
        <label style={{ color: 'var(--text-muted)', fontSize: 13 }}>Filter by host:</label>
        <select
          value={selectedHost}
          onChange={e => setSelectedHost(e.target.value)}
          style={{
            padding: '7px 12px',
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 'var(--radius)',
            color: 'var(--text)',
          }}
        >
          <option value="">All hosts</option>
          {(stats?.hosts ?? []).map(h => (
            <option key={h} value={h}>{h}</option>
          ))}
        </select>
      </div>

      {/* Search */}
      <Section title="Search">
        <div style={{ padding: 16 }}>
          <SearchBar host={selectedHost} />
        </div>
      </Section>

      {/* Recent memories */}
      <Section title="Recent Memories">
        <MemoryTable items={memories} loading={memoriesLoading} error={memoriesError} />
      </Section>

      {/* Recent notes */}
      <Section title="Recent Notes">
        <NoteList items={notes} loading={notesLoading} error={notesError} />
      </Section>
    </div>
  );
}
