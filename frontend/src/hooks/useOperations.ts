/**
 * useOperations - Hook for fetching and polling operations data
 * Ported from ds-app-noc v1
 */

import { useState, useEffect, useCallback, useRef } from 'react';
import type { OperationsData } from '../types';

const POLL_INTERVAL = 30000; // 30 seconds

interface UseOperationsReturn {
  data: OperationsData | null;
  isLoading: boolean;
  error: string | null;
  refresh: () => void;
}

export function useOperations(): UseOperationsReturn {
  const [data, setData] = useState<OperationsData | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const intervalRef = useRef<number | null>(null);

  const fetchOperations = useCallback(async () => {
    try {
      const response = await fetch('/api/operations');
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }
      const result = await response.json();
      // HAL response has data nested, or directly on root
      const operationsData: OperationsData = result.data || result;
      setData(operationsData);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch operations data');
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchOperations();
    intervalRef.current = window.setInterval(fetchOperations, POLL_INTERVAL);
    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, [fetchOperations]);

  return { data, isLoading, error, refresh: fetchOperations };
}
