/**
 * Automation Dashboard Page
 * 
 * Displays automation status, logs, and controls for the DSKanban
 * CI/CD automation system. Provides real-time monitoring of worker
 * activity, queue status, and project-level breakdowns.
 * 
 * Ported from ds-app-noc v1
 * @see Issue #1936: Port ds-app-noc frontend pages to ds-noc-v2
 */

import { useTranslation } from 'react-i18next';
import { AutomationDashboard } from '../../components';

export function AutomationPage() {
  const { t } = useTranslation();

  return (
    <div className="min-h-screen bg-noc-bg">
      <div className="max-w-7xl mx-auto px-4 py-6 space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
              🤖 {t('automation.title', 'Automation Dashboard')}
            </h1>
          </div>
        </div>

        {/* Dashboard Content */}
        <AutomationDashboard />
      </div>
    </div>
  );
}

export default AutomationPage;
