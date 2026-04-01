/**
 * OperationsPanel tests
 * @covers FR-API-007
 */

import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { OperationsPanel } from './OperationsPanel';

const mockOperationsData = {
  _links: {
    self: { href: '/api/operations' },
  },
  data: {
    events: [
      {
        id: 'evt-001',
        timestamp: new Date().toISOString(),
        type: 'deployment',
        severity: 'info',
        service: 'DS Account',
        message: 'Deployment completed successfully',
        status: 'completed',
      },
      {
        id: 'evt-002',
        timestamp: new Date(Date.now() - 15 * 60000).toISOString(),
        type: 'config_change',
        severity: 'info',
        service: 'DS Projects',
        message: 'Feature flag updated',
        status: 'in_progress',
        user: 'lucca',
      },
    ],
    quickActions: [
      {
        id: 'action-1',
        name: 'Clear Cache',
        description: 'Clear CloudFront cache',
        enabled: true,
      },
      {
        id: 'action-2',
        name: 'Restart Lambda',
        description: 'Force cold start',
        enabled: true,
        dangerous: true,
      },
    ],
    scheduleMaintenanceWindows: [],
    systemLoad: {
      requestsPerMinute: 42,
      activeConnections: 8,
      queuedJobs: 0,
      errorRate: 0.02,
    },
  },
};

describe('OperationsPanel', () => {
  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it('renders loading state initially', () => {
    global.fetch = vi.fn(() => new Promise(() => {}));
    render(<OperationsPanel />);
    expect(screen.getByText('Operations Center')).toBeInTheDocument();
  });

  it('renders operations data after loading', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(mockOperationsData),
    });

    render(<OperationsPanel />);

    await waitFor(() => {
      expect(screen.getByText('DS Account')).toBeInTheDocument();
    });

    // Check system load stats
    expect(screen.getByText('42')).toBeInTheDocument();
    expect(screen.getByText('Requests/min')).toBeInTheDocument();
    expect(screen.getByText('0.02%')).toBeInTheDocument();
  });

  it('renders error state on fetch failure', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 500,
      statusText: 'Internal Server Error',
    });

    render(<OperationsPanel />);

    await waitFor(() => {
      expect(screen.getByText(/HTTP 500/)).toBeInTheDocument();
    });

    expect(screen.getByText('Retry')).toBeInTheDocument();
  });

  it('switches between tabs', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(mockOperationsData),
    });

    render(<OperationsPanel />);

    await waitFor(() => {
      expect(screen.getByText('DS Account')).toBeInTheDocument();
    });

    // Click Quick Actions tab
    fireEvent.click(screen.getByText('Quick Actions'));
    expect(screen.getByText('Clear Cache')).toBeInTheDocument();

    // Click Maintenance tab
    fireEvent.click(screen.getByText('Maintenance'));
    expect(screen.getByText('No scheduled maintenance')).toBeInTheDocument();
  });

  it('displays event status badges', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(mockOperationsData),
    });

    render(<OperationsPanel />);

    await waitFor(() => {
      expect(screen.getByText('Completed')).toBeInTheDocument();
      expect(screen.getByText('In Progress')).toBeInTheDocument();
    });
  });

  it('shows badge count for in-progress events', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(mockOperationsData),
    });

    render(<OperationsPanel />);

    await waitFor(() => {
      // Should show badge with count 1 (one in_progress event)
      const badge = screen.getByText('1');
      expect(badge).toBeInTheDocument();
    });
  });

  it('highlights dangerous actions', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(mockOperationsData),
    });

    render(<OperationsPanel />);

    await waitFor(() => {
      expect(screen.getByText('DS Account')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Quick Actions'));
    
    const dangerousAction = screen.getByText('Restart Lambda');
    expect(dangerousAction.closest('button')).toHaveClass('border-red-800');
  });

  it('retries on error when retry button clicked', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
      })
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockOperationsData),
      });

    global.fetch = fetchMock;

    render(<OperationsPanel />);

    await waitFor(() => {
      expect(screen.getByText('Retry')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Retry'));

    await waitFor(() => {
      expect(screen.getByText('DS Account')).toBeInTheDocument();
    });

    expect(fetchMock).toHaveBeenCalledTimes(2);
  });
});
