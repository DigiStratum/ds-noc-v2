/**
 * Hook for HATEOAS navigation using HAL+JSON discovery
 */
import { useState, useEffect, useCallback } from 'react';
import type { DiscoveryResponse, HALLinks } from '../api/hal';
import { getHref, hasLink } from '../api/hal';

interface UseHALNavigationResult {
  /** Discovery document (null until loaded) */
  discovery: DiscoveryResponse | null;
  /** Loading state */
  loading: boolean;
  /** Error if discovery fetch failed */
  error: Error | null;
  /** Get href for a link relation */
  getHref: (rel: string, params?: Record<string, string>) => string | undefined;
  /** Check if a link relation exists */
  hasLink: (rel: string) => boolean;
  /** Refresh the discovery document */
  refresh: () => Promise<void>;
}

const CACHE_KEY = 'hal-discovery';
const CACHE_TTL = 5 * 60 * 1000; // 5 minutes

interface CacheEntry {
  data: DiscoveryResponse;
  timestamp: number;
}

/**
 * Hook for HAL+JSON HATEOAS navigation
 * 
 * @example
 * ```tsx
 * const { discovery, getHref, hasLink } = useHALNavigation();
 * 
 * // Navigate using discovered links
 * const itemsUrl = getHref('ds:items');
 * const itemUrl = getHref('ds:item', { id: '123' });
 * 
 * // Check capabilities
 * if (hasLink('ds:admin')) {
 *   // Show admin UI
 * }
 * ```
 */
export function useHALNavigation(): UseHALNavigationResult {
  const [discovery, setDiscovery] = useState<DiscoveryResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const fetchDiscovery = useCallback(async () => {
    // Check cache first
    try {
      const cached = sessionStorage.getItem(CACHE_KEY);
      if (cached) {
        const entry: CacheEntry = JSON.parse(cached);
        if (Date.now() - entry.timestamp < CACHE_TTL) {
          setDiscovery(entry.data);
          setLoading(false);
          return;
        }
      }
    } catch {
      // Cache miss or invalid, continue to fetch
    }

    try {
      setLoading(true);
      const response = await fetch('/api/discovery', {
        headers: { Accept: 'application/hal+json' },
      });

      if (!response.ok) {
        throw new Error(`Discovery failed: ${response.status}`);
      }

      const data: DiscoveryResponse = await response.json();
      setDiscovery(data);
      setError(null);

      // Cache the response
      const entry: CacheEntry = { data, timestamp: Date.now() };
      sessionStorage.setItem(CACHE_KEY, JSON.stringify(entry));
    } catch (e) {
      setError(e instanceof Error ? e : new Error(String(e)));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchDiscovery();
  }, [fetchDiscovery]);

  const getHrefCallback = useCallback(
    (rel: string, params?: Record<string, string>): string | undefined => {
      return getHref(discovery?._links as HALLinks | undefined, rel, params);
    },
    [discovery]
  );

  const hasLinkCallback = useCallback(
    (rel: string): boolean => {
      return hasLink(discovery?._links as HALLinks | undefined, rel);
    },
    [discovery]
  );

  return {
    discovery,
    loading,
    error,
    getHref: getHrefCallback,
    hasLink: hasLinkCallback,
    refresh: fetchDiscovery,
  };
}

export default useHALNavigation;
