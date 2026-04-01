/**
 * useDashboard - Hook for fetching and polling dashboard state
 * Ported from ds-app-noc v1
 */

import { useState, useEffect, useCallback, useRef } from 'react';
import type { DashboardState, HistoricalDataPoint } from '../types';

const POLL_INTERVAL = 10000; // 10 seconds
const MAX_HISTORY_POINTS = 60; // 10 minutes of data at 10s intervals

interface UseDashboardReturn {
  state: DashboardState | null;
  history: Record<string, HistoricalDataPoint[]>;
  isLoading: boolean;
  error: string | null;
  refresh: () => void;
}

export function useDashboard(): UseDashboardReturn {
  const [state, setState] = useState<DashboardState | null>(null);
  const [history, setHistory] = useState<Record<string, HistoricalDataPoint[]>>({});
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const intervalRef = useRef<number | null>(null);

  const fetchDashboard = useCallback(async () => {
    try {
      const response = await fetch('/api/dashboard');
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }
      const data: DashboardState = await response.json();
      setState(data);
      setError(null);

      // Update history with new data points
      setHistory((prev) => {
        const updated = { ...prev };
        for (const [serviceId, health] of Object.entries(data.services)) {
          if (health) {
            const point: HistoricalDataPoint = {
              timestamp: health.timestamp,
              responseTimeMs: health.responseTimeMs,
              status: health.status,
            };
            const existing = updated[serviceId] || [];
            updated[serviceId] = [...existing.slice(-MAX_HISTORY_POINTS + 1), point];
          }
        }
        return updated;
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch dashboard');
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchDashboard();
    intervalRef.current = window.setInterval(fetchDashboard, POLL_INTERVAL);
    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, [fetchDashboard]);

  return { state, history, isLoading, error, refresh: fetchDashboard };
}
