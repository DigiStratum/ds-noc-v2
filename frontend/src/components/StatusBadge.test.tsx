/**
 * StatusBadge tests
 */

import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { StatusBadge } from './StatusBadge';

describe('StatusBadge', () => {
  it('renders healthy status with success color', () => {
    render(<StatusBadge status="healthy" />);

    const badge = screen.getByRole('status');
    expect(badge).toHaveClass('bg-ds-success');
    expect(badge).toHaveAttribute('aria-label', 'Status: healthy');
  });

  it('renders degraded status with warning color', () => {
    render(<StatusBadge status="degraded" />);

    const badge = screen.getByRole('status');
    expect(badge).toHaveClass('bg-ds-warning');
  });

  it('renders unhealthy status with danger color', () => {
    render(<StatusBadge status="unhealthy" />);

    const badge = screen.getByRole('status');
    expect(badge).toHaveClass('bg-ds-danger');
  });

  it('renders with small size', () => {
    render(<StatusBadge status="healthy" size="sm" />);

    const badge = screen.getByRole('status');
    expect(badge).toHaveClass('w-2', 'h-2');
  });

  it('renders with medium size by default', () => {
    render(<StatusBadge status="healthy" />);

    const badge = screen.getByRole('status');
    expect(badge).toHaveClass('w-3', 'h-3');
  });

  it('renders with large size', () => {
    render(<StatusBadge status="healthy" size="lg" />);

    const badge = screen.getByRole('status');
    expect(badge).toHaveClass('w-4', 'h-4');
  });

  it('has pulse animation', () => {
    render(<StatusBadge status="healthy" />);

    const badge = screen.getByRole('status');
    expect(badge).toHaveClass('animate-pulse');
  });
});
