/**
 * DSKanban Automation API Client
 * 
 * Client for calling DSKanban's automation endpoints from the NOC dashboard.
 * Uses SSO session cookies (credentials: 'include') for cross-app authentication.
 * 
 * Ported from ds-app-noc v1
 * @see Issue #1936: Port ds-app-noc frontend pages to ds-noc-v2
 */

// Production DSKanban URL - uses DS_URLS from @digistratum/ds-core pattern
const DSKANBAN_API_URL = 'https://projects.digistratum.com';

// ============================================
// Type Definitions (matching DSKanban backend)
// ============================================

export interface AutomationStats {
  // Issue counts
  total_queued_issues: number;     // backlog + todo (claimable)
  claimed_issues: number;          // currently claimed (in-progress with claim)
  in_progress_issues: number;      // all in-progress
  blocked_issues: number;          // blocked state
  completed_today: number;         // done in last 24h
  completed_this_week: number;     // done in last 7 days
  failed_attempts: number;         // issues with attempt_count > 0
  
  // Timing stats
  avg_completion_minutes: number;  // avg time from claim to done
  oldest_claim_minutes: number;    // longest active claim
  
  // Per-project breakdown (when project_id=0)
  project_stats?: ProjectAutomationStats[];
}

export interface ProjectAutomationStats {
  project_id: number;
  project_name: string;
  queued_count: number;
  claimed_count: number;
  blocked_count: number;
  completed_today: number;
}

export interface AutomationActivityItem {
  issue_id: number;
  issue_title: string;
  project_id: number;
  project_name: string;
  action: 'claimed' | 'completed' | 'failed' | 'released' | 'blocked' | 'active_claim';
  timestamp: string; // ISO 8601
  details?: string;  // e.g., "attempt 2 of 3"
}

export interface QueueNextIssue {
  id: number;
  title: string;
  priority: number;
  project_id: number;
  project_priority: number;
  type: string;
  state: string;
  workflow_position: number;
}

export interface QueueNextResponse {
  issues: QueueNextIssue[];
  total_available: number;
}

// ============================================
// API Client
// ============================================

interface DSKanbanApiOptions {
  baseUrl?: string;
  timeout?: number;
}

class DSKanbanApiClient {
  private baseUrl: string;
  private timeout: number;

  constructor(options: DSKanbanApiOptions = {}) {
    this.baseUrl = options.baseUrl || DSKANBAN_API_URL;
    this.timeout = options.timeout || 30000; // 30s default
  }

  /**
   * Make a request to DSKanban API with SSO session cookies
   */
  private async request<T>(
    method: string,
    path: string,
    params?: Record<string, string | number | undefined>
  ): Promise<T> {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.timeout);

    try {
      // Build URL with query params
      const url = new URL(`${this.baseUrl}${path}`);
      if (params) {
        Object.entries(params).forEach(([key, value]) => {
          if (value !== undefined && value !== null) {
            url.searchParams.set(key, String(value));
          }
        });
      }

      const response = await fetch(url.toString(), {
        method,
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include', // Critical: include SSO session cookies
        signal: controller.signal,
      });

      clearTimeout(timeoutId);

      if (!response.ok) {
        if (response.status === 401) {
          throw new Error('Unauthorized: DSKanban session expired or not authenticated');
        }
        if (response.status === 403) {
          throw new Error('Forbidden: No access to DSKanban automation API');
        }
        const text = await response.text();
        throw new Error(text || `DSKanban API error: ${response.status}`);
      }

      return response.json();
    } catch (error) {
      clearTimeout(timeoutId);
      
      if (error instanceof Error && error.name === 'AbortError') {
        throw new Error('DSKanban API request timed out');
      }
      throw error;
    }
  }

  // ============================================
  // Automation Dashboard Endpoints
  // ============================================

  /**
   * Get automation statistics for all projects or a specific project
   * @param projectId Optional project ID to filter stats (0 or undefined for all)
   */
  async getAutomationStats(projectId?: number): Promise<AutomationStats> {
    return this.request<AutomationStats>('GET', '/api/automation/stats', {
      project_id: projectId && projectId > 0 ? projectId : undefined,
    });
  }

  /**
   * Get recent automation activity (claims, completions, failures)
   * @param projectId Optional project ID to filter activity
   * @param limit Number of activity items to return (max 100, default 50)
   */
  async getAutomationActivity(
    projectId?: number,
    limit: number = 50
  ): Promise<AutomationActivityItem[]> {
    return this.request<AutomationActivityItem[]>('GET', '/api/automation/activity', {
      project_id: projectId && projectId > 0 ? projectId : undefined,
      limit,
    });
  }

  // ============================================
  // Queue Endpoints (for worker status)
  // ============================================

  /**
   * Get next available queue items (for monitoring worker state)
   * @param projectId Optional project ID filter
   * @param limit Number of items (max 10, default 5)
   */
  async getQueueNext(projectId?: number, limit: number = 5): Promise<QueueNextResponse> {
    return this.request<QueueNextResponse>('GET', '/api/queue/next', {
      project_id: projectId && projectId > 0 ? projectId : undefined,
      limit,
    });
  }

  /**
   * Get blocked tasks in the queue
   */
  async getBlockedTasks(): Promise<QueueNextIssue[]> {
    return this.request<QueueNextIssue[]>('GET', '/api/queue/blocked');
  }
}

// Export singleton instance
export const dskanbanApi = new DSKanbanApiClient();

// Also export class for testing/custom configuration
export { DSKanbanApiClient };
