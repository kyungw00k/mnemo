import { useState, useEffect, useCallback } from 'react';
import { fetchStats, fetchGraph } from '../api/client';
import type { GraphData, StatsData } from '../types';
import { KnowledgeGraph } from '../components/KnowledgeGraph';

export function GraphPage() {
  const [stats, setStats] = useState<StatsData | null>(null);
  const [selectedHost, setSelectedHost] = useState('');
  const [graphData, setGraphData] = useState<GraphData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchStats().then(setStats).catch(() => {/* stats are optional here */});
  }, []);

  const loadGraph = useCallback(async (host: string) => {
    setLoading(true);
    setError(null);
    try {
      const data = await fetchGraph({ host: host || undefined });
      setGraphData(data);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load graph');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadGraph(selectedHost);
  }, [selectedHost, loadGraph]);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      {/* Toolbar */}
      <div style={{
        padding: '12px 24px',
        borderBottom: '1px solid var(--border)',
        display: 'flex',
        alignItems: 'center',
        gap: 16,
        background: 'var(--bg-card)',
        flexShrink: 0,
      }}>
        <span style={{ color: 'var(--text-muted)', fontSize: 13 }}>Host:</span>
        <select
          value={selectedHost}
          onChange={e => setSelectedHost(e.target.value)}
          style={{
            padding: '6px 10px',
            background: 'var(--bg)',
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

        {graphData && (
          <span style={{ color: 'var(--text-dim)', fontSize: 12 }}>
            {graphData.nodes.length} nodes · {graphData.edges.length} edges
          </span>
        )}
      </div>

      {/* Graph */}
      <div style={{ flex: 1, position: 'relative', minHeight: 0 }}>
        <KnowledgeGraph graphData={graphData} loading={loading} error={error} />
      </div>
    </div>
  );
}
