/**
 * Alert types for DS NOC v2
 */

export type AlertType = 'recovery' | 'outage' | 'degradation' | 'change';
export type AlertSeverity = 'critical' | 'warning' | 'info';

export interface Alert {
  id: string;
  serviceId: string;
  serviceName: string;
  timestamp: string;
  type: AlertType;
  severity: AlertSeverity;
  previousStatus: string;
  currentStatus: string;
  message: string;
  latencyMs?: number;
}

export interface AlertsResponse {
  alerts: Alert[];
  count: number;
  since: string;
  _links?: {
    self: { href: string };
  };
}
