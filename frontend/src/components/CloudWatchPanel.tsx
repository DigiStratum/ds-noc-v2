/**
 * CloudWatch Metrics Panel
 * 
 * Displays AWS CloudWatch metrics with interactive charts.
 * Ported from ds-app-noc v1
 * @see Issue #1936: Port ds-app-noc frontend pages to ds-noc-v2
 */

import { useEffect, useState, useCallback } from 'react';
import { LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts';

interface CloudWatchMetric {
  metricName: string;
  namespace: string;
  dimensions: Record<string, string>;
  unit: string;
  datapoints: {
    timestamp: string;
    value: number;
    unit: string;
  }[];
  statistics: {
    average: number;
    maximum: number;
    minimum: number;
    sum: number;
  };
}

interface CloudWatchResponse {
  metrics: CloudWatchMetric[];
  period: string;
  startTime: string;
  endTime: string;
}

interface CloudWatchPanelProps {
  serviceId?: string;
  compact?: boolean;
}

const METRIC_COLORS: Record<string, string> = {
  'CPUUtilization': '#ef4444',
  'MemoryUtilization': '#8b5cf6',
  'NetworkIn': '#3b82f6',
  'NetworkOut': '#06b6d4',
  'Invocations': '#22c55e',
  'Errors': '#ef4444',
  'Duration': '#eab308',
  'Throttles': '#f97316',
  'ConcurrentExecutions': '#6366f1',
  '4XXError': '#eab308',
  '5XXError': '#ef4444',
  'Latency': '#3b82f6',
};

export function CloudWatchPanel({ serviceId, compact = false }: CloudWatchPanelProps) {
  const [metrics, setMetrics] = useState<CloudWatchMetric[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedNamespace, setSelectedNamespace] = useState<string>('all');
  const [timeRange, setTimeRange] = useState<'1h' | '6h' | '24h' | '7d'>('1h');

  const loadMetrics = useCallback(async () => {
    try {
      const params = new URLSearchParams({ range: timeRange });
      if (serviceId) params.set('service', serviceId);
      
      const response = await fetch(`/api/cloudwatch/metrics?${params}`);
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }
      const data: CloudWatchResponse = await response.json();
      setMetrics(data.metrics);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load CloudWatch metrics');
    } finally {
      setIsLoading(false);
    }
  }, [serviceId, timeRange]);

  useEffect(() => {
    loadMetrics();
    const interval = setInterval(loadMetrics, 60000); // Refresh every minute
    return () => clearInterval(interval);
  }, [loadMetrics]);

  const namespaces = [...new Set(metrics.map(m => m.namespace))];
  const filteredMetrics = selectedNamespace === 'all' 
    ? metrics 
    : metrics.filter(m => m.namespace === selectedNamespace);

  // Group metrics by namespace for display
  const groupedMetrics = filteredMetrics.reduce((acc, metric) => {
    const key = metric.namespace;
    if (!acc[key]) acc[key] = [];
    acc[key].push(metric);
    return acc;
  }, {} as Record<string, CloudWatchMetric[]>);

  const formatValue = (value: number, unit: string): string => {
    if (unit === 'Percent') return `${value.toFixed(1)}%`;
    if (unit === 'Bytes') {
      if (value > 1024 * 1024 * 1024) return `${(value / (1024 * 1024 * 1024)).toFixed(1)} GB`;
      if (value > 1024 * 1024) return `${(value / (1024 * 1024)).toFixed(1)} MB`;
      if (value > 1024) return `${(value / 1024).toFixed(1)} KB`;
      return `${value} B`;
    }
    if (unit === 'Milliseconds') return `${value.toFixed(0)} ms`;
    if (unit === 'Count') return value.toLocaleString();
    return value.toFixed(2);
  };

  if (isLoading) {
    return (
      <div className="bg-noc-card border border-noc-border rounded-lg p-6">
        <div className="flex items-center gap-2 mb-4">
          <CloudIcon />
          <h2 className="text-lg font-semibold text-white">CloudWatch Metrics</h2>
        </div>
        <div className="flex items-center justify-center h-40">
          <div className="animate-spin w-8 h-8 border-3 border-ds-primary border-t-transparent rounded-full" />
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-noc-card border border-noc-border rounded-lg p-6">
        <div className="flex items-center gap-2 mb-4">
          <CloudIcon />
          <h2 className="text-lg font-semibold text-white">CloudWatch Metrics</h2>
        </div>
        <div className="text-center text-gray-400 py-8">
          <p>{error}</p>
          <button 
            onClick={loadMetrics}
            className="mt-3 text-ds-primary hover:text-ds-primary-dark text-sm"
          >
            Retry
          </button>
        </div>
      </div>
    );
  }

  if (compact) {
    // Compact view - just show key metrics as stats
    const keyMetrics = metrics.filter(m => 
      ['CPUUtilization', 'MemoryUtilization', 'Errors', 'Latency', '5XXError'].includes(m.metricName)
    );

    return (
      <div className="bg-noc-card border border-noc-border rounded-lg p-4">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <CloudIcon className="w-4 h-4" />
            <h3 className="text-sm font-semibold text-white">CloudWatch</h3>
          </div>
          <span className="text-xs text-gray-500">Last {timeRange}</span>
        </div>
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
          {keyMetrics.slice(0, 4).map(metric => (
            <div key={`${metric.namespace}-${metric.metricName}`} className="text-center">
              <div className="text-lg font-bold text-white">
                {formatValue(metric.statistics.average, metric.unit)}
              </div>
              <div className="text-xs text-gray-400">{metric.metricName}</div>
            </div>
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="bg-noc-card border border-noc-border rounded-lg p-6">
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2">
          <CloudIcon />
          <h2 className="text-lg font-semibold text-white">CloudWatch Metrics</h2>
        </div>
        <div className="flex items-center gap-3">
          {/* Namespace Filter */}
          <select
            value={selectedNamespace}
            onChange={(e) => setSelectedNamespace(e.target.value)}
            className="bg-noc-bg border border-noc-border rounded px-3 py-1.5 text-sm text-gray-300 focus:outline-none focus:border-ds-primary"
          >
            <option value="all">All Namespaces</option>
            {namespaces.map(ns => (
              <option key={ns} value={ns}>{ns.replace('AWS/', '')}</option>
            ))}
          </select>

          {/* Time Range */}
          <div className="flex gap-1 bg-noc-bg rounded p-1">
            {(['1h', '6h', '24h', '7d'] as const).map(range => (
              <button
                key={range}
                onClick={() => setTimeRange(range)}
                className={`px-2 py-1 text-xs rounded transition-colors ${
                  timeRange === range 
                    ? 'bg-ds-primary text-white' 
                    : 'text-gray-400 hover:text-white'
                }`}
              >
                {range}
              </button>
            ))}
          </div>
        </div>
      </div>

      {Object.keys(groupedMetrics).length === 0 ? (
        <div className="text-center text-gray-400 py-12">
          <p>No metrics available</p>
        </div>
      ) : (
        <div className="space-y-6">
          {Object.entries(groupedMetrics).map(([namespace, nsMetrics]) => (
            <div key={namespace}>
              <h3 className="text-sm font-medium text-gray-400 mb-3">
                {namespace.replace('AWS/', '')}
              </h3>
              
              {/* Stats Grid */}
              <div className="grid grid-cols-2 sm:grid-cols-4 lg:grid-cols-6 gap-3 mb-4">
                {nsMetrics.map(metric => (
                  <MetricStatCard
                    key={`${metric.namespace}-${metric.metricName}`}
                    metric={metric}
                    formatValue={formatValue}
                  />
                ))}
              </div>

              {/* Charts for key metrics */}
              {nsMetrics.filter(m => m.datapoints.length >= 5).slice(0, 2).map(metric => (
                <MetricChart key={`chart-${metric.metricName}`} metric={metric} />
              ))}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

interface MetricStatCardProps {
  metric: CloudWatchMetric;
  formatValue: (value: number, unit: string) => string;
}

function MetricStatCard({ metric, formatValue }: MetricStatCardProps) {
  const isError = metric.metricName.toLowerCase().includes('error');
  const isHigh = metric.metricName === 'CPUUtilization' && metric.statistics.average > 80;
  
  return (
    <div className="bg-noc-bg rounded-lg p-3">
      <div className="text-xs text-gray-500 mb-1 truncate" title={metric.metricName}>
        {metric.metricName}
      </div>
      <div 
        className={`text-xl font-bold ${
          (isError && metric.statistics.sum > 0) || isHigh 
            ? 'text-status-unhealthy' 
            : 'text-white'
        }`}
      >
        {formatValue(metric.statistics.average, metric.unit)}
      </div>
      <div className="flex justify-between text-xs text-gray-500 mt-1">
        <span>min: {formatValue(metric.statistics.minimum, metric.unit)}</span>
        <span>max: {formatValue(metric.statistics.maximum, metric.unit)}</span>
      </div>
    </div>
  );
}

interface MetricChartProps {
  metric: CloudWatchMetric;
}

function MetricChart({ metric }: MetricChartProps) {
  const chartData = metric.datapoints.map(dp => ({
    time: new Date(dp.timestamp).toLocaleTimeString('en-US', {
      hour12: false,
      hour: '2-digit',
      minute: '2-digit',
    }),
    value: dp.value,
  }));

  const color = METRIC_COLORS[metric.metricName] || '#3b82f6';

  return (
    <div className="bg-noc-bg rounded-lg p-4 mb-3">
      <div className="flex items-center justify-between mb-3">
        <span className="text-sm text-gray-300">{metric.metricName}</span>
        <span className="text-xs text-gray-500">{metric.unit}</span>
      </div>
      <div className="h-32">
        <ResponsiveContainer width="100%" height="100%">
          <LineChart data={chartData}>
            <XAxis
              dataKey="time"
              stroke="#64748b"
              fontSize={10}
              tickLine={false}
            />
            <YAxis
              stroke="#64748b"
              fontSize={10}
              tickLine={false}
              domain={[0, 'auto']}
            />
            <Tooltip
              contentStyle={{
                backgroundColor: '#1e293b',
                border: '1px solid #334155',
                borderRadius: '4px',
              }}
              labelStyle={{ color: '#94a3b8' }}
              itemStyle={{ color }}
            />
            <Line
              type="monotone"
              dataKey="value"
              stroke={color}
              strokeWidth={2}
              dot={false}
              name={metric.metricName}
            />
          </LineChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}

function CloudIcon({ className = 'w-5 h-5' }: { className?: string }) {
  return (
    <svg className={`${className} text-ds-primary`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 15a4 4 0 004 4h9a5 5 0 10-.1-9.999 5.002 5.002 0 10-9.78 2.096A4.001 4.001 0 003 15z" />
    </svg>
  );
}

export default CloudWatchPanel;
