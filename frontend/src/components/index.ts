/**
 * Common components - TEMPLATE layer
 * Re-exports from @digistratum packages plus template-specific components.
 */

// Template-specific components (not app-specific)
export { LoadingSpinner } from './LoadingSpinner';
export { ErrorBoundary } from './ErrorBoundary';

// Note: Button, Input, Dialog, Card should be installed separately
// (e.g., from shadcn/ui or similar) in derived apps that need them.
