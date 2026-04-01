import { useState, useEffect, useCallback } from 'react';
import { apiClient } from '../api/client';

interface FeatureFlagsResponse {
  flags: Record<string, boolean>;
}

interface UseFeatureFlagsResult {
  flags: Record<string, boolean>;
  isLoading: boolean;
  error: Error | null;
  isEnabled: (flagKey: string) => boolean;
  refresh: () => Promise<void>;
}

/**
 * Hook to evaluate feature flags for the current user context.
 * 
 * Usage:
 *   const { isEnabled, flags, isLoading } = useFeatureFlags();
 *   
 *   if (isEnabled('new-dashboard')) {
 *     return <NewDashboard />;
 *   }
 *   return <OldDashboard />;
 */
export function useFeatureFlags(): UseFeatureFlagsResult {
  const [flags, setFlags] = useState<Record<string, boolean>>({});
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const fetchFlags = useCallback(async () => {
    try {
      setIsLoading(true);
      setError(null);
      
      const response = await apiClient.get<FeatureFlagsResponse>('/api/feature-flags/evaluate');
      setFlags(response.flags || {});
    } catch (err) {
      setError(err instanceof Error ? err : new Error('Failed to fetch feature flags'));
      // Default to empty flags on error
      setFlags({});
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchFlags();
  }, [fetchFlags]);

  const isEnabled = useCallback((flagKey: string): boolean => {
    return flags[flagKey] ?? false;
  }, [flags]);

  return {
    flags,
    isLoading,
    error,
    isEnabled,
    refresh: fetchFlags,
  };
}
