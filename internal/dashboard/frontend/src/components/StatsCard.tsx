interface StatsCardProps {
  label: string;
  value: number | string;
  sub?: string;
}

export function StatsCard({ label, value, sub }: StatsCardProps) {
  return (
    <div style={{
      background: 'var(--bg-card)',
      border: '1px solid var(--border)',
      borderRadius: 'var(--radius)',
      padding: '20px 24px',
      display: 'flex',
      flexDirection: 'column',
      gap: 4,
    }}>
      <span style={{ color: 'var(--text-muted)', fontSize: 12, textTransform: 'uppercase', letterSpacing: '0.05em' }}>
        {label}
      </span>
      <span style={{ fontSize: 32, fontWeight: 700, color: 'var(--text)', lineHeight: 1.2 }}>
        {value}
      </span>
      {sub && (
        <span style={{ color: 'var(--text-dim)', fontSize: 12 }}>{sub}</span>
      )}
    </div>
  );
}
