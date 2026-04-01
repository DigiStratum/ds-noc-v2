/**
 * React hook for DSKanban automation data
 * 
 * Provides automated fetching and caching of automation stats and activity
 * from the DSKanban API for use in NOC dashboard components.
 * 
 * Ported from ds-app-noc v1
 * @see Issue #1936: Port ds-app-noc frontend pages to ds-noc-v2
 */

import { useState, useEffect, useCallback, useRef } from 'react';
import { 
  dskanbanApi, 
  type AutomationStats, 
  type AutomationActivityItem,
  type QueueNextIssue,
} from '../api/dskanban';

interface UseAutomationOptions {
  projectId?: number;
  refreshInterval?: number; // ms, default 30000 (30s)
  activityLimit?: number;   // default 50
  enabled?: boolean;        // default true
}

interface UseAutomationResult {
  stats: AutomationStats | null;
  activity: AutomationActivityItem[];
  isLoading: boolean;
  error: Error | null;
  lastUpdated: Date | null;
  refetch: () => Promise<void>;
}

/**
 * Hook for fetching and auto-refreshing automation data
 */
export function useAutomation(options: UseAutomationOptions = {}): UseAutomationResult {
  const {
    projectId,
    refreshInterval = 30000,
    activityLimit = 50,
    enabled = true,
  } = options;

  const [stats, setStats] = useState<AutomationStats | null>(null);
  const [activity, setActivity] = useState<AutomationActivityItem[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  
  const isMounted = useRef(true);

  const fetchData = useCallback(async () => {
    if (!enabled) return;
    
    try {
      // Fetch both in parallel
      const [statsData, activityData] = await Promise.all([
        dskanbanApi.getAutomationStats(projectId),
        dskanbanApi.getAutomationActivity(projectId, activityLimit),
      ]);
      
      if (isMounted.current) {
        setStats(statsData);
        setActivity(activityData);
        setError(null);
        setLastUpdated(new Date());
        setIsLoading(false);
      }
    } catch (err) {
      if (isMounted.current) {
        setError(err instanceof Error ? err : new Error(String(err)));
        setIsLoading(false);
      }
    }
  }, [projectId, activityLimit, enabled]);

  // Initial fetch
  useEffect(() => {
    isMounted.current = true;
    fetchData();
    
    return () => {
      isMounted.current = false;
    };
  }, [fetchData]);

  // Auto-refresh
  useEffect(() => {
    if (!enabled || refreshInterval <= 0) return;
    
    const intervalId = setInterval(fetchData, refreshInterval);
    return () => clearInterval(intervalId);
  }, [fetchData, refreshInterval, enabled]);

  const refetch = useCallback(async () => {
    setIsLoading(true);
    await fetchData();
  }, [fetchData]);

  return {
    stats,
    activity,
    isLoading,
    error,
    lastUpdated,
    refetch,
  };
}

/**
 * Hook for fetching queue status (next available items)
 */
interface UseAutomationQueueOptions {
  projectId?: number;
  limit?: number;
  refreshInterval?: number;
  enabled?: boolean;
}

interface UseAutomationQueueResult {
  nextItems: QueueNextIssue[];
  totalAvailable: number;
  isLoading: boolean;
  error: Error | null;
  refetch: () => Promise<void>;
}

export function useAutomationQueue(options: UseAutomationQueueOptions = {}): UseAutomationQueueResult {
  const {
    projectId,
    limit = 5,
    refreshInterval = 30000,
    enabled = true,
  } = options;

  const [nextItems, setNextItems] = useState<QueueNextIssue[]>([]);
  const [totalAvailable, setTotalAvailable] = useState(0);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);
  
  const isMounted = useRef(true);

  const fetchData = useCallback(async () => {
    if (!enabled) return;
    
    try {
      const data = await dskanbanApi.getQueueNext(projectId, limit);
      
      if (isMounted.current) {
        setNextItems(data.issues);
        setTotalAvailable(data.total_available);
        setError(null);
        setIsLoading(false);
      }
    } catch (err) {
      if (isMounted.current) {
        setError(err instanceof Error ? err : new Error(String(err)));
        setIsLoading(false);
      }
    }
  }, [projectId, limit, enabled]);

  // Initial fetch
  useEffect(() => {
    isMounted.current = true;
    fetchData();
    
    return () => {
      isMounted.current = false;
    };
  }, [fetchData]);

  useEffect(() => {
    if (!enabled || refreshInterval <= 0) return;
    
    const intervalId = setInterval(fetchData, refreshInterval);
    return () => clearInterval(intervalId);
  }, [fetchData, refreshInterval, enabled]);

  const refetch = useCallback(async () => {
    setIsLoading(true);
    await fetchData();
  }, [fetchData]);

  return {
    nextItems,
    totalAvailable,
    isLoading,
    error,
    refetch,
  };
}
