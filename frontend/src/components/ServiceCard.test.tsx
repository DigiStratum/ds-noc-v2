/**
 * ServiceCard tests
 */

import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ServiceCard } from './ServiceCard';
import type { ServiceHealth } from '../types';

const healthyService: ServiceHealth = {
  status: 'healthy',
  version: '1.2.3',
  uptime: 86400,
  timestamp: '2026-03-31T12:00:00Z',
  service: 'test-service',
  environment: 'production',
  responseTimeMs: 45,
  memory: {
    heapUsedMB: 128,
    heapTotalMB: 256,
    rssMB: 300,
    percentUsed: 50,
  },
  cpu: {
    loadAverage: [0.5, 0.6, 0.7],
    percentUsed: 25,
  },
};

const degradedService: ServiceHealth = {
  ...healthyService,
  status: 'degraded',
  cpu: { loadAverage: [2.5, 2.0, 1.8], percentUsed: 70 },
};

const unhealthyService: ServiceHealth = {
  ...healthyService,
  status: 'unhealthy',
  cpu: { loadAverage: [4.0, 3.5, 3.0], percentUsed: 95 },
};

describe('ServiceCard', () => {
  it('renders service name and status when healthy', () => {
    render(<ServiceCard serviceId="svc-1" health={healthyService} />);

    expect(screen.getByText('test-service')).toBeInTheDocument();
    expect(screen.getByText('healthy')).toBeInTheDocument();
    expect(screen.getByText('1.2.3')).toBeInTheDocument();
    expect(screen.getByText('45ms')).toBeInTheDocument();
    expect(screen.getByText('1d 0h')).toBeInTheDocument();
  });

  it('renders "No data" when health is null', () => {
    render(<ServiceCard serviceId="missing-svc" health={null} />);

    expect(screen.getByText('missing-svc')).toBeInTheDocument();
    expect(screen.getByText('No data')).toBeInTheDocument();
  });

  it('calls onClick when clicked', () => {
    const onClick = vi.fn();
    render(<ServiceCard serviceId="svc-1" health={healthyService} onClick={onClick} />);

    fireEvent.click(screen.getByRole('button'));
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it('supports keyboard navigation', () => {
    const onClick = vi.fn();
    render(<ServiceCard serviceId="svc-1" health={healthyService} onClick={onClick} />);

    fireEvent.keyDown(screen.getByRole('button'), { key: 'Enter' });
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it('renders memory usage bar', () => {
    render(<ServiceCard serviceId="svc-1" health={healthyService} />);

    expect(screen.getByText('Memory')).toBeInTheDocument();
    expect(screen.getByText('128.0MB / 256.0MB')).toBeInTheDocument();
  });

  it('renders CPU usage with appropriate color', () => {
    render(<ServiceCard serviceId="svc-1" health={degradedService} />);

    expect(screen.getByText('CPU')).toBeInTheDocument();
    expect(screen.getByText('70.0%')).toBeInTheDocument();
  });

  it('uses serviceId when service name is not provided', () => {
    const serviceWithoutName = { ...healthyService, service: undefined };
    render(<ServiceCard serviceId="fallback-id" health={serviceWithoutName} />);

    expect(screen.getByText('fallback-id')).toBeInTheDocument();
  });

  it('applies correct status colors', () => {
    const { rerender } = render(<ServiceCard serviceId="svc" health={healthyService} />);
    expect(screen.getByText('healthy')).toHaveClass('text-ds-success');

    rerender(<ServiceCard serviceId="svc" health={degradedService} />);
    expect(screen.getByText('degraded')).toHaveClass('text-ds-warning');

    rerender(<ServiceCard serviceId="svc" health={unhealthyService} />);
    expect(screen.getByText('unhealthy')).toHaveClass('text-ds-danger');
  });
});
