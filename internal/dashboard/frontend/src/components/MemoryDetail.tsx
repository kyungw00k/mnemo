import { useEffect, useState } from 'react';
import { fetchMemoryDetail } from '../api/client';
import type { MemoryItem } from '../types';
import { MarkdownRenderer } from './MarkdownRenderer';

interface MemoryDetailProps {
  memoryId: number | null;
  onClose: () => void;
}

function formatDate(iso: string): string {
  try {
    return new Date(iso).toLocaleString(undefined, {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  } catch {
    return iso;
  }
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

export function MemoryDetail({ memoryId, onClose }: MemoryDetailProps) {
  const [memory, setMemory] = useState<MemoryItem | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!memoryId) {
      setMemory(null);
      setError(null);
      return;
    }

    setLoading(true);
    setError(null);

    fetchMemoryDetail(memoryId)
      .then(setMemory)
      .catch(e => setError(e instanceof Error ? e.message : 'Failed to load memory'))
      .finally(() => setLoading(false));
  }, [memoryId]);

  if (!memoryId) return null;

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(0, 0, 0, 0.5)',
        backdropFilter: 'blur(4px)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 1000,
        padding: 20,
        animation: 'fadeIn 0.2s ease-out',
      }}
      onClick={onClose}
    >
      <div
        style={{
          background: 'var(--bg-card)',
          border: '1px solid var(--border)',
          borderRadius: 12,
          maxWidth: 700,
          width: '100%',
          maxHeight: '80vh',
          overflow: 'auto',
          boxShadow: '0 25px 50px -12px rgba(0, 0, 0, 0.4)',
          animation: 'slideUp 0.3s ease-out',
        }}
        onClick={e => e.stopPropagation()}
      >
        {/* Header */}
        <div style={{
          padding: '20px 24px',
          borderBottom: '1px solid var(--border)',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'flex-start',
          gap: 16,
          background: 'linear-gradient(to bottom, var(--bg-hover), var(--bg-card))',
        }}>
          <div style={{ flex: 1 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 8 }}>
              {memory?.category && (
                <span style={{
                  display: 'inline-block',
                  padding: '3px 10px',
                  borderRadius: 6,
                  fontSize: 12,
                  fontWeight: 600,
                  background: categoryColor(memory.category) + '22',
                  color: categoryColor(memory.category),
                  border: `1px solid ${categoryColor(memory.category)}44`,
                }}>
                  {memory.category}
                </span>
              )}
            </div>
            <h2 style={{
              margin: 0,
              fontSize: 20,
              fontWeight: 600,
              color: 'var(--text)',
              lineHeight: 1.3,
              fontFamily: 'var(--font-mono)',
            }}>
              {memory?.key || (loading ? 'Loading...' : 'Memory')}
            </h2>
          </div>
          <button
            onClick={onClose}
            style={{
              background: 'transparent',
              border: 'none',
              color: 'var(--text-muted)',
              cursor: 'pointer',
              padding: 4,
              borderRadius: 4,
              fontSize: 20,
              lineHeight: 1,
              transition: 'all 0.15s',
            }}
            onMouseEnter={e => {
              e.currentTarget.style.background = 'var(--bg-hover)';
              e.currentTarget.style.color = 'var(--text)';
            }}
            onMouseLeave={e => {
              e.currentTarget.style.background = 'transparent';
              e.currentTarget.style.color = 'var(--text-muted)';
            }}
          >
            ×
          </button>
        </div>

        {/* Content */}
        <div style={{ padding: '24px' }}>
          {loading && (
            <div style={{ textAlign: 'center', padding: 40, color: 'var(--text-muted)' }}>
              Loading memory…
            </div>
          )}

          {error && (
            <div style={{ padding: 20, background: '#fee', border: '1px solid #f88', borderRadius: 8, color: '#c33' }}>
              {error}
            </div>
          )}

          {memory && (
            <div style={{ animation: 'fadeIn 0.3s ease-out' }}>
              {/* Metadata */}
              {memory.metadata && (
                <div style={{ marginBottom: 20 }}>
                  <h4 style={{
                    margin: '0 0 8px 0',
                    fontSize: 12,
                    textTransform: 'uppercase',
                    letterSpacing: '0.05em',
                    color: 'var(--text-muted)',
                  }}>
                    Metadata
                  </h4>
                  <div style={{
                    padding: 12,
                    background: 'var(--bg)',
                    border: '1px solid var(--border)',
                    borderRadius: 6,
                    fontFamily: 'var(--font-mono)',
                    fontSize: 13,
                    color: 'var(--text-muted)',
                    whiteSpace: 'pre-wrap',
                    overflow: 'auto',
                  }}>
                    {memory.metadata}
                  </div>
                </div>
              )}

              {/* Value */}
              <div>
                <h4 style={{
                  margin: '0 0 12px 0',
                  fontSize: 12,
                  textTransform: 'uppercase',
                  letterSpacing: '0.05em',
                  color: 'var(--text-muted)',
                }}>
                  Value
                </h4>
                <MarkdownRenderer content={memory.value} />
              </div>

              {/* Footer metadata */}
              <div style={{
                marginTop: 24,
                paddingTop: 16,
                borderTop: '1px solid var(--border)',
                display: 'flex',
                gap: 20,
                fontSize: 12,
                color: 'var(--text-dim)',
              }}>
                <div>
                  <span style={{ color: 'var(--text-muted)' }}>Host:</span> {memory.host}
                </div>
                <div>
                  <span style={{ color: 'var(--text-muted)' }}>Created:</span> {formatDate(memory.created_at)}
                </div>
                <div>
                  <span style={{ color: 'var(--text-muted)' }}>Updated:</span> {formatDate(memory.updated_at)}
                </div>
              </div>
            </div>
          )}
        </div>
      </div>

      <style>{`
        @keyframes fadeIn {
          from { opacity: 0; }
          to { opacity: 1; }
        }
        @keyframes slideUp {
          from {
            opacity: 0;
            transform: translateY(20px) scale(0.98);
          }
          to {
            opacity: 1;
            transform: translateY(0) scale(1);
          }
        }
      `}</style>
    </div>
  );
}
