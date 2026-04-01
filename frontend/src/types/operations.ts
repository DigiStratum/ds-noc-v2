/**
 * Operations center types for DS NOC v2
 * Ported from ds-app-noc v1
 */

export type EventType = 'deployment' | 'scaling' | 'maintenance' | 'incident' | 'recovery' | 'config_change' | 'alert';
export type EventStatus = 'in_progress' | 'completed' | 'failed';
export type EventSeverity = 'info' | 'warning' | 'error';

export interface SystemEvent {
  id: string;
  timestamp: string;
  type: EventType;
  severity: EventSeverity;
  service: string;
  message: string;
  status: EventStatus;
  user?: string;
}

export interface QuickAction {
  id: string;
  name: string;
  description: string;
  icon?: string;
  service?: string;
  dangerous?: boolean;
  enabled: boolean;
}

export interface MaintenanceWindow {
  id: string;
  service: string;
  startTime: string;
  endTime: string;
  description: string;
}

export interface SystemLoad {
  requestsPerMinute: number;
  activeConnections: number;
  queuedJobs: number;
  errorRate: number;
}

export interface OperationsData {
  events: SystemEvent[];
  quickActions: QuickAction[];
  scheduleMaintenanceWindows: MaintenanceWindow[];
  systemLoad: SystemLoad;
}
