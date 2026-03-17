import { useState, useEffect } from 'react';
import { DashboardPage } from './pages/DashboardPage';
import { GraphPage } from './pages/GraphPage';

type Page = 'dashboard' | 'graph';

function getPageFromHash(): Page {
  const hash = window.location.hash;
  if (hash === '#graph' || hash === '#/graph') return 'graph';
  return 'dashboard';
}

function NavLink({
  label,
  target,
  active,
  onClick,
}: {
  label: string;
  target: string;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <a
      href={`#${target}`}
      onClick={e => { e.preventDefault(); onClick(); }}
      style={{
        padding: '6px 16px',
        borderRadius: 6,
        fontSize: 14,
        fontWeight: active ? 600 : 400,
        color: active ? 'var(--text)' : 'var(--text-muted)',
        background: active ? 'var(--bg-hover)' : 'transparent',
        textDecoration: 'none',
        transition: 'all 0.1s',
        border: active ? '1px solid var(--border)' : '1px solid transparent',
      }}
    >
      {label}
    </a>
  );
}

export default function App() {
  const [page, setPage] = useState<Page>(getPageFromHash);

  useEffect(() => {
    const handler = () => setPage(getPageFromHash());
    window.addEventListener('hashchange', handler);
    return () => window.removeEventListener('hashchange', handler);
  }, []);

  function navigate(target: Page) {
    window.location.hash = target === 'dashboard' ? '' : target;
    setPage(target);
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100vh', overflow: 'hidden' }}>
      {/* Top navigation */}
      <header style={{
        display: 'flex',
        alignItems: 'center',
        padding: '0 24px',
        height: 56,
        borderBottom: '1px solid var(--border)',
        background: 'var(--bg-card)',
        flexShrink: 0,
        gap: 24,
      }}>
        {/* Logo */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="#6366f1" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <circle cx="12" cy="12" r="3" />
            <path d="M6.3 6.3a8 8 0 1 0 11.4 0" />
            <path d="M12 2v4" />
          </svg>
          <span style={{ fontWeight: 700, fontSize: 16, color: 'var(--text)' }}>mnemo</span>
          <span style={{
            fontSize: 10, padding: '1px 6px', borderRadius: 4,
            background: '#6366f122', color: '#6366f1', fontWeight: 600,
          }}>
            dashboard
          </span>
        </div>

        {/* Nav links */}
        <nav style={{ display: 'flex', gap: 4, marginLeft: 8 }}>
          <NavLink
            label="Dashboard"
            target="dashboard"
            active={page === 'dashboard'}
            onClick={() => navigate('dashboard')}
          />
          <NavLink
            label="Knowledge Graph"
            target="graph"
            active={page === 'graph'}
            onClick={() => navigate('graph')}
          />
        </nav>
      </header>

      {/* Page content */}
      <main style={{ flex: 1, overflow: page === 'graph' ? 'hidden' : 'auto', minHeight: 0 }}>
        {page === 'dashboard' ? <DashboardPage /> : <GraphPage />}
      </main>
    </div>
  );
}
