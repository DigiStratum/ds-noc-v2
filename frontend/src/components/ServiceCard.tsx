/**
 * ServiceCard - Compact service health display card
 * Ported from ds-app-noc v1, using DS design tokens
 */

import type { ServiceHealth } from '../types';
import { StatusBadge } from './StatusBadge';

interface ServiceCardProps {
  serviceId: string;
  health: ServiceHealth | null;
  onClick?: () => void;
}

export function ServiceCard({ serviceId, health, onClick }: ServiceCardProps) {
  if (!health) {
    return (
      <div
        className="bg-ds-bg-primary border border-ds-border rounded-lg p-4 cursor-pointer hover:border-ds-text-secondary transition-colors"
        onClick={onClick}
        role="button"
        tabIndex={0}
        onKeyDown={(e) => e.key === 'Enter' && onClick?.()}
      >
        <div className="flex items-center justify-between mb-2">
          <h3 className="text-ds-text-primary font-medium">{serviceId}</h3>
          <span className="w-3 h-3 bg-ds-text-tertiary rounded-full" />
        </div>
        <p className="text-ds-text-tertiary text-sm">No data</p>
      </div>
    );
  }

  const statusColor = {
    healthy: 'text-ds-success',
    degraded: 'text-ds-warning',
    unhealthy: 'text-ds-danger',
  }[health.status];

  return (
    <div
      className="bg-ds-bg-primary border border-ds-border rounded-lg p-4 cursor-pointer hover:border-ds-text-secondary transition-colors"
      onClick={onClick}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => e.key === 'Enter' && onClick?.()}
    >
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-ds-text-primary font-medium">{health.service || serviceId}</h3>
        <StatusBadge status={health.status} />
      </div>

      <div className="space-y-2 text-sm">
        <div className="flex justify-between">
          <span className="text-ds-text-secondary">Status</span>
          <span className={statusColor}>{health.status}</span>
        </div>
        <div className="flex justify-between">
          <span className="text-ds-text-secondary">Response</span>
          <span className="text-ds-text-primary">{health.responseTimeMs}ms</span>
        </div>
        <div className="flex justify-between">
          <span className="text-ds-text-secondary">Version</span>
          <span className="text-ds-text-primary">{health.version}</span>
        </div>
        <div className="flex justify-between">
          <span className="text-ds-text-secondary">Uptime</span>
          <span className="text-ds-text-primary">{formatUptime(health.uptime)}</span>
        </div>
      </div>

      {health.memory && (
        <div className="mt-3 pt-3 border-t border-ds-border">
          <div className="text-xs text-ds-text-secondary mb-1">Memory</div>
          <div className="w-full bg-ds-bg-tertiary rounded-full h-2">
            <div
              className="bg-ds-primary rounded-full h-2 transition-all"
              style={{ width: `${health.memory.percentUsed}%` }}
            />
          </div>
          <div className="text-xs text-ds-text-tertiary mt-1">
            {health.memory.heapUsedMB.toFixed(1)}MB / {health.memory.heapTotalMB.toFixed(1)}MB
          </div>
        </div>
      )}

      {health.cpu && (
        <div className="mt-2">
          <div className="text-xs text-ds-text-secondary mb-1">CPU</div>
          <div className="w-full bg-ds-bg-tertiary rounded-full h-2">
            <div
              className={`rounded-full h-2 transition-all ${
                health.cpu.percentUsed > 80
                  ? 'bg-ds-danger'
                  : health.cpu.percentUsed > 60
                    ? 'bg-ds-warning'
                    : 'bg-ds-success'
              }`}
              style={{ width: `${health.cpu.percentUsed}%` }}
            />
          </div>
          <div className="text-xs text-ds-text-tertiary mt-1">
            {health.cpu.percentUsed.toFixed(1)}%
          </div>
        </div>
      )}
    </div>
  );
}

function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);

  if (days > 0) return `${days}d ${hours}h`;
  if (hours > 0) return `${hours}h ${minutes}m`;
  return `${minutes}m`;
}
