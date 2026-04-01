/**
 * OverviewPanel - System health summary dashboard panel
 * Ported from ds-app-noc v1, using DS design tokens
 */

import type { DashboardState } from '../types';
import { StatusBadge } from './StatusBadge';

interface OverviewPanelProps {
  state: DashboardState;
}

export function OverviewPanel({ state }: OverviewPanelProps) {
  const services = Object.entries(state.services);
  const healthyCount = services.filter(([, h]) => h?.status === 'healthy').length;
  const degradedCount = services.filter(([, h]) => h?.status === 'degraded').length;
  const unhealthyCount = services.filter(([, h]) => h?.status === 'unhealthy').length;

  const totalResponseTime = services
    .filter(([, h]) => h)
    .reduce((sum, [, h]) => sum + (h?.responseTimeMs || 0), 0);
  const avgResponseTime =
    services.length > 0
      ? Math.round(totalResponseTime / services.filter(([, h]) => h).length)
      : 0;

  return (
    <div className="bg-ds-bg-primary border border-ds-border rounded-lg p-6">
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-3">
          <h2 className="text-xl font-semibold text-ds-text-primary">System Overview</h2>
          <StatusBadge status={state.overallStatus} size="lg" />
        </div>
        <div className="text-sm text-ds-text-tertiary">
          Last updated: {new Date(state.lastUpdated).toLocaleTimeString()}
        </div>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
        <StatCard label="Total Services" value={services.length.toString()} />
        <StatCard label="Healthy" value={healthyCount.toString()} color="text-ds-success" />
        <StatCard label="Degraded" value={degradedCount.toString()} color="text-ds-warning" />
        <StatCard label="Unhealthy" value={unhealthyCount.toString()} color="text-ds-danger" />
        <StatCard label="Avg Response" value={`${avgResponseTime}ms`} />
      </div>
    </div>
  );
}

interface StatCardProps {
  label: string;
  value: string;
  color?: string;
}

function StatCard({ label, value, color = 'text-ds-text-primary' }: StatCardProps) {
  return (
    <div className="text-center">
      <div className={`text-2xl font-bold ${color}`}>{value}</div>
      <div className="text-sm text-ds-text-tertiary">{label}</div>
    </div>
  );
}
