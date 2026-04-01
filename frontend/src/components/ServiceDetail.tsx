/**
 * ServiceDetail - Full service health detail modal
 * Ported from ds-app-noc v1, using DS design tokens
 */

import type { ServiceHealth } from '../types';

interface ServiceDetailProps {
  serviceId: string;
  health: ServiceHealth;
  onClose: () => void;
}

export function ServiceDetail({ serviceId, health, onClose }: ServiceDetailProps) {
  return (
    <div
      className="fixed inset-0 bg-ds-bg-tertiary/50 dark:bg-black/50 flex items-center justify-center z-50"
      onClick={(e) => e.target === e.currentTarget && onClose()}
      role="dialog"
      aria-modal="true"
      aria-labelledby="service-detail-title"
    >
      <div className="bg-ds-bg-primary border border-ds-border rounded-lg w-full max-w-3xl max-h-[90vh] overflow-auto m-4 shadow-lg">
        <div className="flex items-center justify-between p-4 border-b border-ds-border">
          <h2 id="service-detail-title" className="text-xl font-semibold text-ds-text-primary">
            {health.service || serviceId}
          </h2>
          <button
            onClick={onClose}
            className="text-ds-text-secondary hover:text-ds-text-primary transition-colors"
            aria-label="Close dialog"
          >
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div className="p-4 space-y-6">
          {/* Status Section */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <InfoCard label="Status" value={health.status} status={health.status} />
            <InfoCard label="Version" value={health.version} />
            <InfoCard label="Environment" value={health.environment || 'unknown'} />
            <InfoCard label="Response Time" value={`${health.responseTimeMs}ms`} />
          </div>

          {/* Resource Utilization */}
          {(health.memory || health.cpu) && (
            <div>
              <h3 className="text-ds-text-primary font-medium mb-3">Resource Utilization</h3>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {health.memory && (
                  <div className="bg-ds-bg-secondary border border-ds-border rounded-lg p-4">
                    <div className="text-sm text-ds-text-secondary mb-2">Memory</div>
                    <div className="w-full bg-ds-bg-tertiary rounded-full h-3">
                      <div
                        className="bg-ds-primary rounded-full h-3 transition-all"
                        style={{ width: `${health.memory.percentUsed}%` }}
                      />
                    </div>
                    <div className="flex justify-between text-xs text-ds-text-tertiary mt-2">
                      <span>Heap: {health.memory.heapUsedMB.toFixed(1)}MB / {health.memory.heapTotalMB.toFixed(1)}MB</span>
                      <span>RSS: {health.memory.rssMB.toFixed(1)}MB</span>
                    </div>
                  </div>
                )}
                {health.cpu && (
                  <div className="bg-ds-bg-secondary border border-ds-border rounded-lg p-4">
                    <div className="text-sm text-ds-text-secondary mb-2">CPU</div>
                    <div className="w-full bg-ds-bg-tertiary rounded-full h-3">
                      <div
                        className={`rounded-full h-3 transition-all ${
                          health.cpu.percentUsed > 80
                            ? 'bg-ds-danger'
                            : health.cpu.percentUsed > 60
                              ? 'bg-ds-warning'
                              : 'bg-ds-success'
                        }`}
                        style={{ width: `${health.cpu.percentUsed}%` }}
                      />
                    </div>
                    <div className="flex justify-between text-xs text-ds-text-tertiary mt-2">
                      <span>Current: {health.cpu.percentUsed.toFixed(1)}%</span>
                      <span>Load: {health.cpu.loadAverage.map((l) => l.toFixed(2)).join(' / ')}</span>
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Connections */}
          {health.connections && (
            <div>
              <h3 className="text-ds-text-primary font-medium mb-3">Connections</h3>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {health.connections.database && (
                  <div className="bg-ds-bg-secondary border border-ds-border rounded-lg p-4">
                    <div className="text-sm text-ds-text-secondary mb-2">Database Pool</div>
                    <div className="grid grid-cols-3 gap-2 text-center">
                      <div>
                        <div className="text-lg font-bold text-ds-success">{health.connections.database.active}</div>
                        <div className="text-xs text-ds-text-tertiary">Active</div>
                      </div>
                      <div>
                        <div className="text-lg font-bold text-ds-text-secondary">{health.connections.database.idle}</div>
                        <div className="text-xs text-ds-text-tertiary">Idle</div>
                      </div>
                      <div>
                        <div className="text-lg font-bold text-ds-text-primary">{health.connections.database.max}</div>
                        <div className="text-xs text-ds-text-tertiary">Max</div>
                      </div>
                    </div>
                  </div>
                )}
                {health.connections.http && (
                  <div className="bg-ds-bg-secondary border border-ds-border rounded-lg p-4">
                    <div className="text-sm text-ds-text-secondary mb-2">HTTP Connections</div>
                    <div className="grid grid-cols-2 gap-2 text-center">
                      <div>
                        <div className="text-lg font-bold text-ds-success">{health.connections.http.active}</div>
                        <div className="text-xs text-ds-text-tertiary">Active</div>
                      </div>
                      <div>
                        <div className="text-lg font-bold text-ds-text-secondary">{health.connections.http.pending}</div>
                        <div className="text-xs text-ds-text-tertiary">Pending</div>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Dependency Checks */}
          {health.checks && Object.keys(health.checks).length > 0 && (
            <div>
              <h3 className="text-ds-text-primary font-medium mb-3">Dependency Checks</h3>
              <div className="space-y-2">
                {Object.entries(health.checks).map(([name, check]) => (
                  <div key={name} className="flex items-center justify-between bg-ds-bg-secondary border border-ds-border rounded-lg p-3">
                    <div className="flex items-center gap-3">
                      <span
                        className={`w-2 h-2 rounded-full ${
                          check.status === 'healthy'
                            ? 'bg-ds-success'
                            : check.status === 'degraded'
                              ? 'bg-ds-warning'
                              : 'bg-ds-danger'
                        }`}
                      />
                      <span className="text-ds-text-primary">{name}</span>
                    </div>
                    <div className="flex items-center gap-4 text-sm">
                      {check.latencyMs !== undefined && (
                        <span className="text-ds-text-secondary">{check.latencyMs}ms</span>
                      )}
                      <span
                        className={
                          check.status === 'healthy'
                            ? 'text-ds-success'
                            : check.status === 'degraded'
                              ? 'text-ds-warning'
                              : 'text-ds-danger'
                        }
                      >
                        {check.status}
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

interface InfoCardProps {
  label: string;
  value: string;
  status?: 'healthy' | 'degraded' | 'unhealthy';
}

function InfoCard({ label, value, status }: InfoCardProps) {
  const color = status
    ? {
        healthy: 'text-ds-success',
        degraded: 'text-ds-warning',
        unhealthy: 'text-ds-danger',
      }[status]
    : 'text-ds-text-primary';

  return (
    <div className="bg-ds-bg-secondary border border-ds-border rounded-lg p-3 text-center">
      <div className={`text-lg font-bold ${color}`}>{value}</div>
      <div className="text-xs text-ds-text-tertiary">{label}</div>
    </div>
  );
}
