/**
 * LoadingSpinner - TEMPLATE layer component
 * Updated: v0.2.0 - Added accessible loading text
 */
export function LoadingSpinner() {
  return (
    <div className="flex items-center justify-center p-4">
      <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
      <span className="sr-only">Loading...</span>
    </div>
  );
}
