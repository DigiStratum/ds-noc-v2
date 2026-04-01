/**
 * OperationsPanel - Operations center dashboard panel
 * Ported from ds-app-noc v1, using DS design tokens
 */

import { useState } from 'react';
import { useOperations } from '../hooks';
import type { SystemEvent, EventType, EventStatus } from '../types';

type TabType = 'events' | 'actions' | 'maintenance';

export function OperationsPanel() {
  const { data, isLoading, error, refresh } = useOperations();
  const [activeTab, setActiveTab] = useState<TabType>('events');

  const eventTypeIcons: Record<EventType, string> = {
    deployment: '🚀',
    scaling: '📈',
    maintenance: '🔧',
    incident: '🚨',
    recovery: '✅',
    config_change: '⚙️',
    alert: '🔔',
  };

  const eventTypeColors: Record<EventType, string> = {
    deployment: 'text-blue-400',
    scaling: 'text-purple-400',
    maintenance: 'text-ds-warning',
    incident: 'text-ds-danger',
    recovery: 'text-ds-success',
    config_change: 'text-ds-info',
    alert: 'text-ds-warning',
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

  if (isLoading) {
    return (
      <div className="bg-ds-bg-primary border border-ds-border rounded-lg p-6">
        <h2 className="text-lg font-semibold text-ds-text-primary mb-4">Operations Center</h2>
        <div className="flex items-center justify-center h-40">
          <div className="animate-spin w-8 h-8 border-2 border-ds-primary border-t-transparent rounded-full" />
        </div>
      </div>
    );
  }

  if (error || !data) {
    return (
      <div className="bg-ds-bg-primary border border-ds-border rounded-lg p-6">
        <h2 className="text-lg font-semibold text-ds-text-primary mb-4">Operations Center</h2>
        <div className="text-center text-ds-text-secondary py-8">
          <p>{error || 'No data available'}</p>
          <button 
            onClick={refresh}
            className="mt-3 text-ds-primary hover:text-ds-primary-dark text-sm"
          >
            Retry
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-ds-bg-primary border border-ds-border rounded-lg">
      {/* Header with system load stats */}
      <div className="p-4 border-b border-ds-border">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-ds-text-primary">Operations Center</h2>
          <span className="text-xs text-ds-text-tertiary">Auto-refreshes every 30s</span>
        </div>
        
        {/* System Load Stats */}
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
          <SystemStat 
            label="Requests/min" 
            value={data.systemLoad.requestsPerMinute.toLocaleString()} 
          />
          <SystemStat 
            label="Active Connections" 
            value={data.systemLoad.activeConnections.toLocaleString()} 
          />
          <SystemStat 
            label="Queued Jobs" 
            value={data.systemLoad.queuedJobs.toLocaleString()}
            alert={data.systemLoad.queuedJobs > 100}
          />
          <SystemStat 
            label="Error Rate" 
            value={`${data.systemLoad.errorRate.toFixed(2)}%`}
            alert={data.systemLoad.errorRate > 1}
          />
        </div>
      </div>

      {/* Tab Navigation */}
      <div className="flex border-b border-ds-border">
        <TabButton 
          active={activeTab === 'events'} 
          onClick={() => setActiveTab('events')}
          badge={data.events.filter(e => e.status === 'in_progress').length}
        >
          Recent Events
        </TabButton>
        <TabButton 
          active={activeTab === 'actions'} 
          onClick={() => setActiveTab('actions')}
        >
          Quick Actions
        </TabButton>
        <TabButton 
          active={activeTab === 'maintenance'} 
          onClick={() => setActiveTab('maintenance')}
          badge={data.scheduleMaintenanceWindows.length}
        >
          Maintenance
        </TabButton>
      </div>

      {/* Tab Content */}
      <div className="p-4">
        {activeTab === 'events' && (
          <div className="space-y-2 max-h-64 overflow-y-auto">
            {data.events.length === 0 ? (
              <p className="text-ds-text-tertiary text-center py-4">No recent events</p>
            ) : (
              data.events.map(event => (
                <div 
                  key={event.id}
                  className="flex items-start gap-3 p-3 bg-ds-bg-secondary rounded-lg"
                >
                  <span className="text-xl">{eventTypeIcons[event.type] || '📋'}</span>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center justify-between gap-2">
                      <span className={`font-medium ${eventTypeColors[event.type] || 'text-ds-text-primary'}`}>
                        {event.service}
                      </span>
                      <span className="text-xs text-ds-text-tertiary">{formatTime(event.timestamp)}</span>
                    </div>
                    <p className="text-sm text-ds-text-secondary mt-0.5">{event.message}</p>
                    <div className="flex items-center gap-2 mt-1">
                      <StatusPill status={event.status} />
                      {event.user && (
                        <span className="text-xs text-ds-text-tertiary">by {event.user}</span>
                      )}
                    </div>
                  </div>
                </div>
              ))
            )}
          </div>
        )}

        {activeTab === 'actions' && (
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            {data.quickActions.map(action => (
              <button
                key={action.id}
                disabled={!action.enabled}
                className={`p-3 rounded-lg text-left transition-colors ${
                  action.dangerous 
                    ? 'bg-red-900/20 border border-red-800 hover:bg-red-900/30' 
                    : 'bg-ds-bg-secondary hover:bg-ds-bg-tertiary'
                } ${!action.enabled ? 'opacity-50 cursor-not-allowed' : ''}`}
                onClick={() => {
                  if (action.enabled) {
                    // eslint-disable-next-line no-alert
                    alert(`Action: ${action.name} - This would trigger the action in production`);
                  }
                }}
              >
                <div className={`font-medium ${action.dangerous ? 'text-ds-danger' : 'text-ds-text-primary'}`}>
                  {action.name}
                </div>
                <div className="text-xs text-ds-text-tertiary mt-1">{action.description}</div>
                {action.service && (
                  <div className="text-xs text-ds-text-tertiary mt-1">Service: {action.service}</div>
                )}
              </button>
            ))}
          </div>
        )}

        {activeTab === 'maintenance' && (
          <div className="space-y-3">
            {data.scheduleMaintenanceWindows.length === 0 ? (
              <p className="text-ds-text-tertiary text-center py-4">No scheduled maintenance</p>
            ) : (
              data.scheduleMaintenanceWindows.map(window => (
                <div key={window.id} className="p-3 bg-ds-bg-secondary rounded-lg">
                  <div className="flex items-center justify-between">
                    <span className="font-medium text-ds-warning">{window.service}</span>
                    <span className="text-xs bg-ds-warning/20 text-ds-warning px-2 py-0.5 rounded">
                      Scheduled
                    </span>
                  </div>
                  <p className="text-sm text-ds-text-secondary mt-1">{window.description}</p>
                  <div className="text-xs text-ds-text-tertiary mt-2">
                    {new Date(window.startTime).toLocaleString()} - {new Date(window.endTime).toLocaleTimeString()}
                  </div>
                </div>
              ))
            )}
          </div>
        )}
      </div>
    </div>
  );
}

interface SystemStatProps {
  label: string;
  value: string;
  alert?: boolean;
}

function SystemStat({ label, value, alert }: SystemStatProps) {
  return (
    <div className="bg-ds-bg-secondary rounded-lg p-3 text-center">
      <div className={`text-xl font-bold ${alert ? 'text-ds-danger' : 'text-ds-text-primary'}`}>
        {value}
      </div>
      <div className="text-xs text-ds-text-tertiary">{label}</div>
    </div>
  );
}

interface TabButtonProps {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
  badge?: number;
}

function TabButton({ active, onClick, children, badge }: TabButtonProps) {
  return (
    <button
      onClick={onClick}
      className={`px-4 py-3 text-sm font-medium transition-colors relative ${
        active 
          ? 'text-ds-primary border-b-2 border-ds-primary -mb-px' 
          : 'text-ds-text-tertiary hover:text-ds-text-primary'
      }`}
    >
      {children}
      {badge !== undefined && badge > 0 && (
        <span className="ml-2 px-1.5 py-0.5 bg-ds-primary text-white text-xs rounded-full">
          {badge}
        </span>
      )}
    </button>
  );
}

interface StatusPillProps {
  status: EventStatus;
}

function StatusPill({ status }: StatusPillProps) {
  const styles: Record<EventStatus, string> = {
    in_progress: 'bg-blue-900/30 text-blue-400',
    completed: 'bg-ds-success/20 text-ds-success',
    failed: 'bg-ds-danger/20 text-ds-danger',
  };

  const labels: Record<EventStatus, string> = {
    in_progress: 'In Progress',
    completed: 'Completed',
    failed: 'Failed',
  };

  return (
    <span className={`text-xs px-2 py-0.5 rounded ${styles[status]}`}>
      {labels[status]}
    </span>
  );
}
