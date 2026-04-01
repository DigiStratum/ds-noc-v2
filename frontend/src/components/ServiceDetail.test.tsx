/**
 * ServiceDetail tests
 */

import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ServiceDetail } from './ServiceDetail';
import type { ServiceHealth, HealthCheck } from '../types';

const fullService: ServiceHealth = {
  status: 'healthy',
  version: '2.0.0',
  uptime: 172800,
  timestamp: '2026-03-31T12:00:00Z',
  service: 'api-gateway',
  environment: 'production',
  responseTimeMs: 35,
  memory: {
    heapUsedMB: 200,
    heapTotalMB: 512,
    rssMB: 600,
    percentUsed: 39,
  },
  cpu: {
    loadAverage: [1.2, 1.5, 1.3],
    percentUsed: 45,
  },
  connections: {
    database: { active: 5, idle: 10, max: 20 },
    http: { active: 100, pending: 5 },
  },
  checks: {
    database: { status: 'healthy', latencyMs: 5 },
    cache: { status: 'healthy', latencyMs: 2 },
    queue: { status: 'degraded', latencyMs: 150 },
  },
};

describe('ServiceDetail', () => {
  it('renders service name in header', () => {
    render(<ServiceDetail serviceId="svc-1" health={fullService} onClose={vi.fn()} />);

    expect(screen.getByText('api-gateway')).toBeInTheDocument();
  });

  it('renders status info cards', () => {
    render(<ServiceDetail serviceId="svc-1" health={fullService} onClose={vi.fn()} />);

    expect(screen.getByText('Status')).toBeInTheDocument();
    expect(screen.getByText('Version')).toBeInTheDocument();
    expect(screen.getByText('Environment')).toBeInTheDocument();
    expect(screen.getByText('Response Time')).toBeInTheDocument();
    expect(screen.getByText('2.0.0')).toBeInTheDocument();
    expect(screen.getByText('production')).toBeInTheDocument();
    expect(screen.getByText('35ms')).toBeInTheDocument();
  });

  it('renders memory utilization', () => {
    render(<ServiceDetail serviceId="svc-1" health={fullService} onClose={vi.fn()} />);

    expect(screen.getByText('Resource Utilization')).toBeInTheDocument();
    expect(screen.getByText(/Heap: 200.0MB \/ 512.0MB/)).toBeInTheDocument();
    expect(screen.getByText(/RSS: 600.0MB/)).toBeInTheDocument();
  });

  it('renders CPU utilization', () => {
    render(<ServiceDetail serviceId="svc-1" health={fullService} onClose={vi.fn()} />);

    expect(screen.getByText(/Current: 45.0%/)).toBeInTheDocument();
    expect(screen.getByText(/Load: 1.20 \/ 1.50 \/ 1.30/)).toBeInTheDocument();
  });

  it('renders connection stats', () => {
    render(<ServiceDetail serviceId="svc-1" health={fullService} onClose={vi.fn()} />);

    expect(screen.getByText('Connections')).toBeInTheDocument();
    expect(screen.getByText('Database Pool')).toBeInTheDocument();
    expect(screen.getByText('HTTP Connections')).toBeInTheDocument();
    // active db connections and http pending are both 5, use getAllByText
    expect(screen.getAllByText('5').length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText('100')).toBeInTheDocument(); // active http
  });

  it('renders dependency checks', () => {
    render(<ServiceDetail serviceId="svc-1" health={fullService} onClose={vi.fn()} />);

    expect(screen.getByText('Dependency Checks')).toBeInTheDocument();
    expect(screen.getByText('database')).toBeInTheDocument();
    expect(screen.getByText('cache')).toBeInTheDocument();
    expect(screen.getByText('queue')).toBeInTheDocument();
    expect(screen.getByText('5ms')).toBeInTheDocument();
    expect(screen.getByText('150ms')).toBeInTheDocument();
  });

  it('calls onClose when close button clicked', () => {
    const onClose = vi.fn();
    render(<ServiceDetail serviceId="svc-1" health={fullService} onClose={onClose} />);

    fireEvent.click(screen.getByLabelText('Close dialog'));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('calls onClose when clicking overlay', () => {
    const onClose = vi.fn();
    render(<ServiceDetail serviceId="svc-1" health={fullService} onClose={onClose} />);

    fireEvent.click(screen.getByRole('dialog'));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('does not render resource section when no memory/cpu', () => {
    const minimalService: ServiceHealth = {
      status: 'healthy',
      version: '1.0.0',
      uptime: 3600,
      timestamp: '2026-03-31T12:00:00Z',
      responseTimeMs: 50,
    };
    render(<ServiceDetail serviceId="svc-1" health={minimalService} onClose={vi.fn()} />);

    expect(screen.queryByText('Resource Utilization')).not.toBeInTheDocument();
  });

  it('applies correct status colors in info cards', () => {
    render(<ServiceDetail serviceId="svc-1" health={fullService} onClose={vi.fn()} />);

    // The status value "healthy" should have success color
    const statusValues = screen.getAllByText('healthy');
    expect(statusValues.some((el) => el.classList.contains('text-ds-success'))).toBe(true);
  });
});
