import { useEffect, useState } from 'react';
import { fetchNoteDetail } from '../api/client';
import type { NoteItem } from '../types';

interface NoteDetailProps {
  noteId: number | null;
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

export function NoteDetail({ noteId, onClose }: NoteDetailProps) {
  const [note, setNote] = useState<NoteItem | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!noteId) {
      setNote(null);
      setError(null);
      return;
    }

    setLoading(true);
    setError(null);

    fetchNoteDetail(noteId)
      .then(setNote)
      .catch(e => setError(e instanceof Error ? e.message : 'Failed to load note'))
      .finally(() => setLoading(false));
  }, [noteId]);

  if (!noteId) return null;

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
            <h2 style={{
              margin: 0,
              fontSize: 22,
              fontWeight: 600,
              color: 'var(--text)',
              lineHeight: 1.3,
            }}>
              {note?.title || (loading ? 'Loading...' : 'Note')}
            </h2>
            {note?.project && (
              <span style={{
                display: 'inline-block',
                marginTop: 8,
                fontSize: 12,
                padding: '3px 10px',
                borderRadius: 6,
                background: 'var(--accent)',
                color: 'white',
                fontWeight: 500,
              }}>
                {note.project}
              </span>
            )}
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
              Loading note…
            </div>
          )}

          {error && (
            <div style={{ padding: 20, background: '#fee', border: '1px solid #f88', borderRadius: 8, color: '#c33' }}>
              {error}
            </div>
          )}

          {note && (
            <div style={{ animation: 'fadeIn 0.3s ease-out' }}>
              {/* Tags */}
              {note.tags && (
                <div style={{ marginBottom: 20, display: 'flex', flexWrap: 'wrap', gap: 8 }}>
                  {note.tags.split(',').map(tag => (
                    <span
                      key={tag}
                      style={{
                        fontSize: 12,
                        padding: '4px 10px',
                        borderRadius: 20,
                        background: 'var(--bg)',
                        border: '1px solid var(--border)',
                        color: 'var(--accent)',
                      }}
                    >
                      #{tag.trim()}
                    </span>
                  ))}
                </div>
              )}

              {/* Content */}
              <div style={{
                lineHeight: 1.7,
                color: 'var(--text)',
                whiteSpace: 'pre-wrap',
                fontSize: 15,
              }}>
                {note.content}
              </div>

              {/* Metadata */}
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
                  <span style={{ color: 'var(--text-muted)' }}>Host:</span> {note.host}
                </div>
                <div>
                  <span style={{ color: 'var(--text-muted)' }}>Created:</span> {formatDate(note.created_at)}
                </div>
                <div>
                  <span style={{ color: 'var(--text-muted)' }}>Updated:</span> {formatDate(note.updated_at)}
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
