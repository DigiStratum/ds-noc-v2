/**
 * NOC Dashboard Page
 * 
 * Main NOC dashboard with service grid, alerts, operations panels.
 * Ported from ds-app-noc v1
 * @see Issue #1936: Port ds-app-noc frontend pages to ds-noc-v2
 */

import { useState } from 'react';
import { useDashboard } from '../../hooks/useDashboard';
import type { ServiceHealth } from '../../types';
import {
  OverviewPanel,
  AlertsPanel,
  CloudWatchPanel,
  OperationsPanel,
  ServiceCard,
  ServiceDetail,
  ResponseTimeChart,
} from '../../components';

export function NocDashboardPage() {
  const { state, history, isLoading, error, refresh } = useDashboard();
  const [selectedService, setSelectedService] = useState<{ id: string; health: ServiceHealth } | null>(null);

  if (isLoading && !state) {
    return (
      <div className="min-h-screen bg-noc-bg flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin w-12 h-12 border-4 border-ds-primary border-t-transparent rounded-full mx-auto" />
          <p className="mt-4 text-gray-400">Loading dashboard...</p>
        </div>
      </div>
    );
  }

  if (error && !state) {
    return (
      <div className="min-h-screen bg-noc-bg flex items-center justify-center">
        <div className="text-center">
          <p className="text-red-400 mb-4">{error}</p>
          <button
            onClick={refresh}
            className="px-4 py-2 bg-ds-primary text-white rounded hover:bg-ds-primary/90 transition-colors"
          >
            Retry
          </button>
        </div>
      </div>
    );
  }

  if (!state) return null;

  const serviceEntries = Object.entries(state.services);

  return (
    <div className="min-h-screen bg-noc-bg">
      <div className="max-w-7xl mx-auto px-4 py-6 space-y-6">
        {/* Overview Panel */}
        <OverviewPanel state={state} />

        {/* Main Content Grid */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Services Grid - Left 2 columns */}
          <div className="lg:col-span-2 space-y-6">
            {/* Service Cards */}
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
              {serviceEntries.map(([serviceId, health]) => (
                <ServiceCard
                  key={serviceId}
                  serviceId={serviceId}
                  health={health}
                  onClick={() => health && setSelectedService({ id: serviceId, health })}
                />
              ))}
            </div>

            {/* Response Time Charts */}
            {Object.entries(history)
              .filter(([, data]) => data.length >= 5)
              .slice(0, 2)
              .map(([serviceId, data]) => (
                <ResponseTimeChart key={serviceId} serviceId={serviceId} data={data} />
              ))}

            {/* CloudWatch Panel - Full Width */}
            <CloudWatchPanel />
          </div>

          {/* Right Column - Alerts & Operations */}
          <div className="space-y-6">
            <AlertsPanel />
            <OperationsPanel />
          </div>
        </div>
      </div>

      {/* Service Detail Modal */}
      {selectedService && (
        <ServiceDetail
          serviceId={selectedService.id}
          health={selectedService.health}
          onClose={() => setSelectedService(null)}
        />
      )}
    </div>
  );
}

export default NocDashboardPage;
