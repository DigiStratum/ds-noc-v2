/**
 * Response Time Chart Component
 * 
 * Displays response time trends for a service with avg/p95 stats.
 * Ported from ds-app-noc v1
 * @see Issue #1936: Port ds-app-noc frontend pages to ds-noc-v2
 */

import { LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer, ReferenceLine } from 'recharts';
import type { HistoricalDataPoint } from '../types';

interface ResponseTimeChartProps {
  serviceId: string;
  data: HistoricalDataPoint[];
}

export function ResponseTimeChart({ serviceId, data }: ResponseTimeChartProps) {
  const chartData = data.map(point => ({
    time: new Date(point.timestamp).toLocaleTimeString('en-US', { 
      hour12: false, 
      hour: '2-digit', 
      minute: '2-digit' 
    }),
    responseTime: point.responseTimeMs,
    status: point.status,
  }));

  // Calculate avg and p95
  const responseTimes = data.map(d => d.responseTimeMs).sort((a, b) => a - b);
  const avg = responseTimes.length > 0 
    ? Math.round(responseTimes.reduce((a, b) => a + b, 0) / responseTimes.length)
    : 0;
  const p95 = responseTimes.length > 0 
    ? responseTimes[Math.floor(responseTimes.length * 0.95)] 
    : 0;

  return (
    <div className="bg-noc-card border border-noc-border rounded-lg p-4">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-white font-medium">{serviceId} - Response Time</h3>
        <div className="flex gap-4 text-sm">
          <span className="text-gray-400">Avg: <span className="text-white">{avg}ms</span></span>
          <span className="text-gray-400">P95: <span className="text-white">{p95}ms</span></span>
        </div>
      </div>
      
      <div className="h-48">
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
              unit="ms"
            />
            <Tooltip 
              contentStyle={{ 
                backgroundColor: '#1e293b', 
                border: '1px solid #334155',
                borderRadius: '4px'
              }}
              labelStyle={{ color: '#94a3b8' }}
              itemStyle={{ color: '#3b82f6' }}
            />
            <ReferenceLine y={200} stroke="#ef4444" strokeDasharray="5 5" label={{ value: 'SLA', fill: '#ef4444', fontSize: 10 }} />
            <Line 
              type="monotone" 
              dataKey="responseTime" 
              stroke="#3b82f6" 
              strokeWidth={2}
              dot={false}
              name="Response Time"
            />
          </LineChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}

export default ResponseTimeChart;
