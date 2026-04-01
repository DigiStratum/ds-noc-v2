import type { HTMLAttributes, ReactNode } from 'react';
import type { AlertType, AlertSeverity } from '../../types';

/**
 * Props for the AlertsPanel component.
 */
export interface AlertsPanelProps extends HTMLAttributes<HTMLDivElement> {
  /** Additional CSS class names */
  className?: string;
  
  /** Child elements */
  children?: ReactNode;

  /** Maximum number of alerts to fetch (default: 20) */
  limit?: number;

  /** Lookback period in hours (default: 24) */
  hours?: number;
}

/**
 * Severity color mappings using DS design tokens
 */
export const severityColors: Record<AlertSeverity, string> = {
  critical: 'bg-ds-danger',
  warning: 'bg-ds-warning',
  info: 'bg-ds-success',
};

export const severityTextColors: Record<AlertSeverity, string> = {
  critical: 'text-ds-danger',
  warning: 'text-ds-warning',
  info: 'text-ds-success',
};

/**
 * Alert type icons
 */
export const typeIcons: Record<AlertType, string> = {
  recovery: '✓',
  outage: '✕',
  degradation: '⚠',
  change: '→',
};
