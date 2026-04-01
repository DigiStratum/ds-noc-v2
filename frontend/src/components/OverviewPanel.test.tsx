/**
 * OverviewPanel tests
 */

import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { OverviewPanel } from './OverviewPanel';
import type { DashboardState } from '../types';

const mockDashboardState: DashboardState = {
  services: {
    dsaccount: {
      status: 'healthy',
      version: '1.0.0',
      uptime: 86400,
      timestamp: '2026-03-31T20:00:00Z',
      service: 'DS Account',
      responseTimeMs: 150,
    },
    dskanban: {
      status: 'degraded',
      version: '2.0.0',
      uptime: 43200,
      timestamp: '2026-03-31T20:00:00Z',
      service: 'DS Projects',
      responseTimeMs: 350,
    },
    developer: {
      status: 'unhealthy',
      version: '1.5.0',
      uptime: 0,
      timestamp: '2026-03-31T20:00:00Z',
      service: 'DS Developer',
      responseTimeMs: 5000,
    },
  },
  lastUpdated: '2026-03-31T20:00:00Z',
  overallStatus: 'unhealthy',
};

describe('OverviewPanel', () => {
  it('renders system overview title', () => {
    render(<OverviewPanel state={mockDashboardState} />);
    expect(screen.getByText('System Overview')).toBeInTheDocument();
  });

  it('displays correct service counts', () => {
    render(<OverviewPanel state={mockDashboardState} />);

    // Verify labels are present
    expect(screen.getByText('Total Services')).toBeInTheDocument();
    expect(screen.getByText('Healthy')).toBeInTheDocument();
    expect(screen.getByText('Degraded')).toBeInTheDocument();
    expect(screen.getByText('Unhealthy')).toBeInTheDocument();

    // Check total services shows 3
    const totalLabel = screen.getByText('Total Services');
    const totalCard = totalLabel.parentElement;
    expect(totalCard?.querySelector('.text-2xl')?.textContent).toBe('3');
  });

  it('calculates average response time', () => {
    render(<OverviewPanel state={mockDashboardState} />);

    // (150 + 350 + 5000) / 3 = 1833ms
    expect(screen.getByText('1833ms')).toBeInTheDocument();
    expect(screen.getByText('Avg Response')).toBeInTheDocument();
  });

  it('displays last updated time', () => {
    render(<OverviewPanel state={mockDashboardState} />);

    // Should contain "Last updated:" text
    expect(screen.getByText(/Last updated:/)).toBeInTheDocument();
  });

  it('handles empty services gracefully', () => {
    const emptyState: DashboardState = {
      services: {},
      lastUpdated: '2026-03-31T20:00:00Z',
      overallStatus: 'healthy',
    };

    render(<OverviewPanel state={emptyState} />);

    // Verify 0 counts are displayed - use getAllByText since there will be multiple
    const zeroElements = screen.getAllByText('0');
    expect(zeroElements.length).toBeGreaterThan(0);
    expect(screen.getByText('0ms')).toBeInTheDocument(); // Avg response with no services
  });

  it('handles null services in the map', () => {
    const stateWithNull: DashboardState = {
      services: {
        dsaccount: {
          status: 'healthy',
          version: '1.0.0',
          uptime: 86400,
          timestamp: '2026-03-31T20:00:00Z',
          service: 'DS Account',
          responseTimeMs: 100,
        },
        dskanban: null as unknown as typeof mockDashboardState.services.dskanban,
      },
      lastUpdated: '2026-03-31T20:00:00Z',
      overallStatus: 'degraded',
    };

    render(<OverviewPanel state={stateWithNull} />);

    // Should not crash and should count correctly
    expect(screen.getByText('System Overview')).toBeInTheDocument();
  });
});
