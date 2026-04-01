/**
 * AlertsPanel - Recent alerts dashboard panel
 * Ported from ds-app-noc v1, using DS design tokens
 */

import { useState } from 'react';
import { useAlerts } from '../../hooks';
import type { AlertsPanelProps, severityColors, severityTextColors, typeIcons } from './types';

/**
 * AlertsPanel Component
 *
 * Displays recent service alerts with expandable detail view.
 * Polls for updates every 30 seconds.
 *
 * @example
 * ```tsx
 * <AlertsPanel limit={20} hours={24} />
 * ```
 */
export function AlertsPanel({ className, limit = 20, hours = 24, ...props }: AlertsPanelProps) {
  const { alerts, isLoading, error, refresh } = useAlerts({ limit, hours });
  const [isExpanded, setIsExpanded] = useState(true);

  const severityColors = {
    critical: 'bg-ds-danger',
    warning: 'bg-ds-warning',
    info: 'bg-ds-success',
  };

  const severityTextColors = {
    critical: 'text-ds-danger',
    warning: 'text-ds-warning',
    info: 'text-ds-success',
  };

  const typeIcons = {
    recovery: '✓',
    outage: '✕',
    degradation: '⚠',
    change: '→',
  };

  const formatTime = (timestamp: string) => {
    const date = new Date(timestamp);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);

    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    return date.toLocaleDateString();
  };

  return (
    <div
      className={`bg-ds-bg-primary border border-ds-border rounded-lg ${className || ''}`}
      data-testid="alerts-panel"
      {...props}
    >
      <div 
        className="flex items-center justify-between p-4 cursor-pointer hover:bg-ds-bg-secondary transition-colors rounded-t-lg"
        onClick={() => setIsExpanded(!isExpanded)}
        role="button"
        aria-expanded={isExpanded}
        tabIndex={0}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            setIsExpanded(!isExpanded);
          }
        }}
      >
        <div className="flex items-center gap-3">
          <h2 className="text-lg font-semibold text-ds-text-primary">Recent Alerts</h2>
          {alerts.length > 0 && (
            <span className="px-2 py-0.5 bg-ds-bg-tertiary text-ds-text-secondary text-sm rounded-full">
              {alerts.length}
            </span>
          )}
        </div>
        <div className="flex items-center gap-4">
          {alerts.filter(a => a.severity === 'critical').length > 0 && (
            <span className="flex items-center gap-1 text-ds-danger text-sm">
              <span className="w-2 h-2 rounded-full bg-ds-danger animate-pulse" />
              {alerts.filter(a => a.severity === 'critical').length} critical
            </span>
          )}
          <svg 
            className={`w-5 h-5 text-ds-text-tertiary transform transition-transform ${isExpanded ? 'rotate-180' : ''}`}
            fill="none" 
            stroke="currentColor" 
            viewBox="0 0 24 24"
            aria-hidden="true"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
          </svg>
        </div>
      </div>

      {isExpanded && (
        <div className="border-t border-ds-border">
          {isLoading ? (
            <div className="p-4 text-center">
              <div className="animate-spin w-6 h-6 border-2 border-ds-primary border-t-transparent rounded-full mx-auto" />
            </div>
          ) : error ? (
            <div className="p-4 text-center text-ds-text-secondary">
              <p>{error}</p>
              <button 
                onClick={refresh}
                className="mt-2 text-ds-primary hover:text-ds-primary-dark text-sm"
              >
                Retry
              </button>
            </div>
          ) : alerts.length === 0 ? (
            <div className="p-4 text-center text-ds-text-secondary">
              <p>No alerts in the last {hours} hours</p>
              <p className="text-sm mt-1">All systems operating normally</p>
            </div>
          ) : (
            <div className="max-h-80 overflow-y-auto">
              {alerts.map((alert) => (
                <div 
                  key={alert.id}
                  className="flex items-start gap-3 p-3 border-b border-ds-border last:border-b-0 hover:bg-ds-bg-secondary transition-colors"
                  data-testid="alert-item"
                >
                  <div className={`w-8 h-8 rounded-full flex items-center justify-center ${severityColors[alert.severity]} bg-opacity-20`}>
                    <span className={severityTextColors[alert.severity]}>
                      {typeIcons[alert.type]}
                    </span>
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center justify-between gap-2">
                      <span className="font-medium text-ds-text-primary truncate">{alert.serviceName}</span>
                      <span className="text-xs text-ds-text-tertiary whitespace-nowrap">{formatTime(alert.timestamp)}</span>
                    </div>
                    <p className="text-sm text-ds-text-secondary mt-0.5">{alert.message}</p>
                    <div className="flex items-center gap-3 mt-1 text-xs">
                      <span className={severityTextColors[alert.severity]}>
                        {alert.severity.toUpperCase()}
                      </span>
                      {alert.latencyMs && (
                        <span className="text-ds-text-tertiary">{alert.latencyMs}ms</span>
                      )}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export default AlertsPanel;
