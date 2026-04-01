import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import AlertsPanel from './AlertsPanel';

// Mock the useAlerts hook
vi.mock('../../hooks', () => ({
  useAlerts: vi.fn(),
}));

import { useAlerts } from '../../hooks';

describe('AlertsPanel', () => {
  const mockRefresh = vi.fn();
  
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.clearAllMocks();
  });

  it('renders loading state', () => {
    (useAlerts as ReturnType<typeof vi.fn>).mockReturnValue({
      alerts: [],
      isLoading: true,
      error: null,
      refresh: mockRefresh,
    });

    render(<AlertsPanel />);
    expect(screen.getByTestId('alerts-panel')).toBeInTheDocument();
    expect(screen.getByText('Recent Alerts')).toBeInTheDocument();
  });

  it('renders empty state when no alerts', () => {
    (useAlerts as ReturnType<typeof vi.fn>).mockReturnValue({
      alerts: [],
      isLoading: false,
      error: null,
      refresh: mockRefresh,
    });

    render(<AlertsPanel hours={24} />);
    expect(screen.getByText('No alerts in the last 24 hours')).toBeInTheDocument();
    expect(screen.getByText('All systems operating normally')).toBeInTheDocument();
  });

  it('renders error state', () => {
    (useAlerts as ReturnType<typeof vi.fn>).mockReturnValue({
      alerts: [],
      isLoading: false,
      error: 'Failed to fetch alerts',
      refresh: mockRefresh,
    });

    render(<AlertsPanel />);
    expect(screen.getByText('Failed to fetch alerts')).toBeInTheDocument();
    expect(screen.getByText('Retry')).toBeInTheDocument();
  });

  it('calls refresh on retry button click', () => {
    (useAlerts as ReturnType<typeof vi.fn>).mockReturnValue({
      alerts: [],
      isLoading: false,
      error: 'Network error',
      refresh: mockRefresh,
    });

    render(<AlertsPanel />);
    fireEvent.click(screen.getByText('Retry'));
    expect(mockRefresh).toHaveBeenCalled();
  });

  it('renders alerts list', () => {
    const mockAlerts = [
      {
        id: '1',
        serviceId: 'svc-1',
        serviceName: 'API Gateway',
        timestamp: new Date().toISOString(),
        type: 'outage' as const,
        severity: 'critical' as const,
        previousStatus: 'healthy',
        currentStatus: 'unhealthy',
        message: 'Service is down',
      },
      {
        id: '2',
        serviceId: 'svc-2',
        serviceName: 'Database',
        timestamp: new Date(Date.now() - 3600000).toISOString(),
        type: 'recovery' as const,
        severity: 'info' as const,
        previousStatus: 'degraded',
        currentStatus: 'healthy',
        message: 'Service recovered',
        latencyMs: 150,
      },
    ];

    (useAlerts as ReturnType<typeof vi.fn>).mockReturnValue({
      alerts: mockAlerts,
      isLoading: false,
      error: null,
      refresh: mockRefresh,
    });

    render(<AlertsPanel />);
    
    expect(screen.getByText('API Gateway')).toBeInTheDocument();
    expect(screen.getByText('Service is down')).toBeInTheDocument();
    expect(screen.getByText('Database')).toBeInTheDocument();
    expect(screen.getByText('Service recovered')).toBeInTheDocument();
    expect(screen.getByText('CRITICAL')).toBeInTheDocument();
    expect(screen.getByText('150ms')).toBeInTheDocument();
  });

  it('shows critical alert count badge', () => {
    const mockAlerts = [
      {
        id: '1',
        serviceId: 'svc-1',
        serviceName: 'API Gateway',
        timestamp: new Date().toISOString(),
        type: 'outage' as const,
        severity: 'critical' as const,
        previousStatus: 'healthy',
        currentStatus: 'unhealthy',
        message: 'Service is down',
      },
    ];

    (useAlerts as ReturnType<typeof vi.fn>).mockReturnValue({
      alerts: mockAlerts,
      isLoading: false,
      error: null,
      refresh: mockRefresh,
    });

    render(<AlertsPanel />);
    expect(screen.getByText('1 critical')).toBeInTheDocument();
  });

  it('toggles expansion on header click', () => {
    (useAlerts as ReturnType<typeof vi.fn>).mockReturnValue({
      alerts: [],
      isLoading: false,
      error: null,
      refresh: mockRefresh,
    });

    render(<AlertsPanel />);
    
    // Initially expanded
    expect(screen.getByText('All systems operating normally')).toBeInTheDocument();
    
    // Click to collapse
    fireEvent.click(screen.getByRole('button'));
    expect(screen.queryByText('All systems operating normally')).not.toBeInTheDocument();
    
    // Click to expand again
    fireEvent.click(screen.getByRole('button'));
    expect(screen.getByText('All systems operating normally')).toBeInTheDocument();
  });

  it('passes custom className', () => {
    (useAlerts as ReturnType<typeof vi.fn>).mockReturnValue({
      alerts: [],
      isLoading: false,
      error: null,
      refresh: mockRefresh,
    });

    render(<AlertsPanel className="custom-class" />);
    expect(screen.getByTestId('alerts-panel')).toHaveClass('custom-class');
  });
});
