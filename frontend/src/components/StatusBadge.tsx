/**
 * StatusBadge - Animated status indicator
 * Ported from ds-app-noc v1, using DS design tokens
 */

interface StatusBadgeProps {
  status: 'healthy' | 'degraded' | 'unhealthy';
  size?: 'sm' | 'md' | 'lg';
}

export function StatusBadge({ status, size = 'md' }: StatusBadgeProps) {
  const colors = {
    healthy: 'bg-ds-success',
    degraded: 'bg-ds-warning',
    unhealthy: 'bg-ds-danger',
  };

  const sizes = {
    sm: 'w-2 h-2',
    md: 'w-3 h-3',
    lg: 'w-4 h-4',
  };

  return (
    <span
      className={`inline-block rounded-full ${colors[status]} ${sizes[size]} animate-pulse`}
      role="status"
      aria-label={`Status: ${status}`}
    />
  );
}
