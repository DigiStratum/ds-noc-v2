/**
 * useAlerts - Hook for fetching and polling alerts data
 * Ported from ds-app-noc v1
 */

import { useState, useEffect, useCallback, useRef } from 'react';
import type { Alert, AlertsResponse } from '../types';

const POLL_INTERVAL = 30000; // 30 seconds

interface UseAlertsOptions {
  limit?: number;
  hours?: number;
}

interface UseAlertsReturn {
  alerts: Alert[];
  isLoading: boolean;
  error: string | null;
  refresh: () => void;
}

export function useAlerts(options: UseAlertsOptions = {}): UseAlertsReturn {
  const { limit = 20, hours = 24 } = options;
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const intervalRef = useRef<number | null>(null);

  const fetchAlerts = useCallback(async () => {
    try {
      const params = new URLSearchParams({
        limit: String(limit),
        hours: String(hours),
      });
      const response = await fetch(`/api/alerts?${params}`);
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }
      const result: AlertsResponse = await response.json();
      setAlerts(result.alerts);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch alerts');
    } finally {
      setIsLoading(false);
    }
  }, [limit, hours]);

  useEffect(() => {
    fetchAlerts();
    intervalRef.current = window.setInterval(fetchAlerts, POLL_INTERVAL);
    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, [fetchAlerts]);

  return { alerts, isLoading, error, refresh: fetchAlerts };
}
