/**
 * AutomationDashboard Component
 * 
 * Displays automation statistics, activity feed, and project breakdown
 * for monitoring DSKanban worker automation from NOC.
 * 
 * Ported from ds-app-noc v1
 * @see Issue #1936: Port ds-app-noc frontend pages to ds-noc-v2
 */

import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useAutomation } from '../hooks/useAutomation';
import type { 
  AutomationActivityItem,
  ProjectAutomationStats,
} from '../api/dskanban';

// ============================================
// Helper Components
// ============================================

interface StatCardProps {
  title: string;
  value: string | number;
  subtitle?: string;
  icon: string;
  color?: 'blue' | 'green' | 'yellow' | 'red' | 'purple' | 'gray';
}

function StatCard({ title, value, subtitle, icon, color = 'blue' }: StatCardProps) {
  const colors: Record<string, string> = {
    blue: 'bg-blue-50 text-blue-700 border-blue-200 dark:bg-blue-900/20 dark:text-blue-300 dark:border-blue-800',
    green: 'bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-300 dark:border-green-800',
    yellow: 'bg-yellow-50 text-yellow-700 border-yellow-200 dark:bg-yellow-900/20 dark:text-yellow-300 dark:border-yellow-800',
    red: 'bg-red-50 text-red-700 border-red-200 dark:bg-red-900/20 dark:text-red-300 dark:border-red-800',
    purple: 'bg-purple-50 text-purple-700 border-purple-200 dark:bg-purple-900/20 dark:text-purple-300 dark:border-purple-800',
    gray: 'bg-gray-50 text-gray-700 border-gray-200 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-700',
  };

  return (
    <div className={`rounded-lg border p-4 ${colors[color]}`}>
      <div className="flex items-center gap-3">
        <div className="text-2xl">{icon}</div>
        <div>
          <div className="text-2xl font-bold">{value}</div>
          <div className="text-sm font-medium">{title}</div>
          {subtitle && <div className="text-xs opacity-75">{subtitle}</div>}
        </div>
      </div>
    </div>
  );
}

interface ActivityItemComponentProps {
  item: AutomationActivityItem;
}

function ActivityItemComponent({ item }: ActivityItemComponentProps) {
  const actionConfig: Record<string, { icon: string; color: string; label: string }> = {
    claimed: { icon: '🎯', color: 'text-blue-600 dark:text-blue-400', label: 'Claimed' },
    active_claim: { icon: '⚡', color: 'text-purple-600 dark:text-purple-400', label: 'Working' },
    completed: { icon: '✅', color: 'text-green-600 dark:text-green-400', label: 'Completed' },
    blocked: { icon: '🚫', color: 'text-red-600 dark:text-red-400', label: 'Blocked' },
    released: { icon: '↩️', color: 'text-yellow-600 dark:text-yellow-400', label: 'Released' },
    failed: { icon: '❌', color: 'text-red-600 dark:text-red-400', label: 'Failed' },
  };

  const config = actionConfig[item.action] || { icon: '📝', color: 'text-gray-600 dark:text-gray-400', label: item.action };
  const timeAgo = formatTimeAgo(new Date(item.timestamp));

  return (
    <div className="flex items-start gap-3 py-2 border-b border-gray-100 dark:border-gray-700 last:border-0">
      <div className="text-xl">{config.icon}</div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className={`font-medium ${config.color}`}>{config.label}</span>
          <span className="text-gray-400">·</span>
          <span className="text-sm text-gray-500 dark:text-gray-400">{timeAgo}</span>
        </div>
        <div className="text-sm text-gray-700 dark:text-gray-300 truncate">
          <span className="font-medium">#{item.issue_id}</span> {item.issue_title}
        </div>
        <div className="text-xs text-gray-500 dark:text-gray-400">
          {item.project_name}
          {item.details && <span className="ml-2 text-orange-600 dark:text-orange-400">({item.details})</span>}
        </div>
      </div>
    </div>
  );
}

interface ProjectStatsTableProps {
  projectStats: ProjectAutomationStats[];
}

function ProjectStatsTable({ projectStats }: ProjectStatsTableProps) {
  if (!projectStats || projectStats.length === 0) return null;

  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg border shadow-sm overflow-hidden">
      <div className="px-4 py-3 border-b bg-gray-50 dark:bg-gray-700/50 dark:border-gray-700">
        <h3 className="font-semibold text-gray-700 dark:text-gray-200">📊 Per-Project Breakdown</h3>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 dark:bg-gray-700/30">
            <tr>
              <th className="px-4 py-2 text-left font-medium text-gray-600 dark:text-gray-300">Project</th>
              <th className="px-4 py-2 text-right font-medium text-gray-600 dark:text-gray-300">Queued</th>
              <th className="px-4 py-2 text-right font-medium text-gray-600 dark:text-gray-300">Claimed</th>
              <th className="px-4 py-2 text-right font-medium text-gray-600 dark:text-gray-300">Blocked</th>
              <th className="px-4 py-2 text-right font-medium text-gray-600 dark:text-gray-300">Done Today</th>
            </tr>
          </thead>
          <tbody>
            {projectStats.map((ps) => (
              <tr key={ps.project_id} className="border-t dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700/30">
                <td className="px-4 py-2 font-medium text-gray-900 dark:text-gray-100">{ps.project_name}</td>
                <td className="px-4 py-2 text-right">
                  <span className={ps.queued_count > 0 ? 'text-blue-600 dark:text-blue-400 font-medium' : 'text-gray-400'}>
                    {ps.queued_count}
                  </span>
                </td>
                <td className="px-4 py-2 text-right">
                  <span className={ps.claimed_count > 0 ? 'text-purple-600 dark:text-purple-400 font-medium' : 'text-gray-400'}>
                    {ps.claimed_count}
                  </span>
                </td>
                <td className="px-4 py-2 text-right">
                  <span className={ps.blocked_count > 0 ? 'text-red-600 dark:text-red-400 font-medium' : 'text-gray-400'}>
                    {ps.blocked_count}
                  </span>
                </td>
                <td className="px-4 py-2 text-right">
                  <span className={ps.completed_today > 0 ? 'text-green-600 dark:text-green-400 font-medium' : 'text-gray-400'}>
                    {ps.completed_today}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

// ============================================
// Utility Functions
// ============================================

function formatTimeAgo(date: Date): string {
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  const minutes = Math.floor(diff / 60000);
  const hours = Math.floor(diff / 3600000);
  const days = Math.floor(diff / 86400000);

  if (minutes < 1) return 'just now';
  if (minutes < 60) return `${minutes}m ago`;
  if (hours < 24) return `${hours}h ago`;
  if (days < 7) return `${days}d ago`;
  return date.toLocaleDateString();
}

function formatMinutes(minutes: number): string {
  if (minutes < 1) return '<1m';
  if (minutes < 60) return `${Math.round(minutes)}m`;
  const hours = minutes / 60;
  if (hours < 24) return `${hours.toFixed(1)}h`;
  const days = hours / 24;
  return `${days.toFixed(1)}d`;
}

// ============================================
// Main Component
// ============================================

interface AutomationDashboardProps {
  /** Optional project ID to filter by */
  projectId?: number;
}

export function AutomationDashboard({ projectId: initialProjectId }: AutomationDashboardProps) {
  const { t } = useTranslation();
  const [selectedProject] = useState<number | undefined>(initialProjectId);
  const [autoRefresh, setAutoRefresh] = useState(true);

  const { 
    stats, 
    activity, 
    isLoading, 
    error, 
    lastUpdated, 
    refetch 
  } = useAutomation({
    projectId: selectedProject,
    refreshInterval: autoRefresh ? 10000 : 0, // 10 seconds when enabled
    activityLimit: 30,
    enabled: true,
  });

  if (isLoading && !stats) {
    return (
      <div className="p-6 flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto"></div>
          <p className="mt-2 text-gray-500 dark:text-gray-400">
            {t('automation.loading', 'Loading automation data...')}
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <p className="text-sm text-gray-500 dark:text-gray-400">
            {t('automation.subtitle', 'Real-time DSKanban worker automation monitoring')}
            {lastUpdated && (
              <span className="ml-2">
                · Updated {formatTimeAgo(lastUpdated)}
              </span>
            )}
          </p>
        </div>
        <div className="flex items-center gap-3">
          {/* Auto-refresh toggle */}
          <label className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400">
            <input
              type="checkbox"
              checked={autoRefresh}
              onChange={(e) => setAutoRefresh(e.target.checked)}
              className="rounded border-gray-300 dark:border-gray-600"
            />
            {t('automation.autoRefresh', 'Auto-refresh')}
          </label>
          
          {/* Manual refresh */}
          <button
            onClick={() => refetch()}
            disabled={isLoading}
            className="px-3 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 text-sm transition-colors"
          >
            🔄 {t('common.refresh', 'Refresh')}
          </button>
        </div>
      </div>

      {/* Error Display */}
      {error && (
        <div className="p-3 bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300 rounded-lg border border-red-200 dark:border-red-800">
          <span className="font-medium">Error:</span> {error.message}
        </div>
      )}

      {stats && (
        <>
          {/* Stats Grid - Row 1 */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <StatCard
              title="Queued Tasks"
              value={stats.total_queued_issues}
              subtitle="Available for workers"
              icon="📋"
              color="blue"
            />
            <StatCard
              title="Active Claims"
              value={stats.claimed_issues}
              subtitle={stats.oldest_claim_minutes > 0 ? `Oldest: ${formatMinutes(stats.oldest_claim_minutes)}` : 'No active claims'}
              icon="⚡"
              color="purple"
            />
            <StatCard
              title="In Progress"
              value={stats.in_progress_issues}
              subtitle="All work in flight"
              icon="🔧"
              color="yellow"
            />
            <StatCard
              title="Blocked"
              value={stats.blocked_issues}
              subtitle={stats.blocked_issues > 0 ? 'Needs attention' : 'All clear'}
              icon="🚫"
              color={stats.blocked_issues > 0 ? 'red' : 'gray'}
            />
          </div>

          {/* Stats Grid - Row 2 */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <StatCard
              title="Done Today"
              value={stats.completed_today}
              icon="✅"
              color="green"
            />
            <StatCard
              title="Done This Week"
              value={stats.completed_this_week}
              icon="📈"
              color="green"
            />
            <StatCard
              title="Avg Completion"
              value={formatMinutes(stats.avg_completion_minutes)}
              subtitle="Claim → Done"
              icon="⏱️"
              color="gray"
            />
            <StatCard
              title="Retry Needed"
              value={stats.failed_attempts}
              subtitle={stats.failed_attempts > 0 ? 'Issues with failures' : 'No failures'}
              icon="🔁"
              color={stats.failed_attempts > 0 ? 'yellow' : 'gray'}
            />
          </div>

          {/* Per-project breakdown */}
          {!selectedProject && stats.project_stats && stats.project_stats.length > 0 && (
            <ProjectStatsTable projectStats={stats.project_stats} />
          )}
        </>
      )}

      {/* Activity Feed */}
      <div className="bg-white dark:bg-gray-800 rounded-lg border dark:border-gray-700 shadow-sm">
        <div className="px-4 py-3 border-b bg-gray-50 dark:bg-gray-700/50 dark:border-gray-700 flex justify-between items-center">
          <h3 className="font-semibold text-gray-700 dark:text-gray-200">
            📜 {t('automation.activity.title', 'Recent Activity')}
          </h3>
          <span className="text-xs text-gray-500 dark:text-gray-400">
            {activity.length} {activity.length === 1 ? 'event' : 'events'}
          </span>
        </div>
        <div className="max-h-96 overflow-y-auto">
          {activity.length === 0 ? (
            <div className="p-8 text-center text-gray-500 dark:text-gray-400">
              {t('automation.activity.noActivity', 'No recent activity')}
            </div>
          ) : (
            <div className="px-4">
              {activity.map((item, idx) => (
                <ActivityItemComponent key={`${item.issue_id}-${item.action}-${idx}`} item={item} />
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Worker Pool Info (placeholder for ASG integration) */}
      <div className="p-4 bg-gray-50 dark:bg-gray-800/50 rounded-lg border border-dashed border-gray-300 dark:border-gray-600">
        <div className="flex items-center gap-2 text-gray-600 dark:text-gray-400">
          <span className="text-xl">🏭</span>
          <span className="font-medium">{t('automation.workerPool.title', 'Worker Pool')}</span>
        </div>
        <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
          {t('automation.workerPool.pendingIntegration', 'ASG worker scaling integration pending.')} 
          {' '}
          <code className="bg-gray-200 dark:bg-gray-700 px-1 rounded text-xs">/api/queue/claim</code>
        </p>
      </div>
    </div>
  );
}

export default AutomationDashboard;
