/**
 * Common components - TEMPLATE layer
 * Re-exports from @digistratum packages plus template-specific components.
 */

// Template-specific components (not app-specific)
export { LoadingSpinner } from './LoadingSpinner';
export { ErrorBoundary } from './ErrorBoundary';

// NOC-specific components
export { StatusBadge } from './StatusBadge';
export { ServiceCard } from './ServiceCard';
export { ServiceDetail } from './ServiceDetail';
export { OverviewPanel } from './OverviewPanel';
export { OperationsPanel } from './OperationsPanel';

// Note: Button, Input, Dialog, Card should be installed separately
// (e.g., from shadcn/ui or similar) in derived apps that need them.
export { AlertsPanel, type AlertsPanelProps } from './AlertsPanel';
