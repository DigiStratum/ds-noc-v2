/**
 * Service health monitoring types for DS NOC v2
 * Ported from ds-app-noc v1
 */

export interface HealthCheck {
  status: 'healthy' | 'degraded' | 'unhealthy';
  latencyMs?: number;
  message?: string;
}

export interface MemoryStats {
  heapUsedMB: number;
  heapTotalMB: number;
  rssMB: number;
  percentUsed: number;
}

export interface CpuStats {
  loadAverage: [number, number, number];
  percentUsed: number;
}

export interface ConnectionStats {
  database?: {
    active: number;
    idle: number;
    max: number;
  };
  http?: {
    active: number;
    pending: number;
  };
}

export interface ServiceHealth {
  status: 'healthy' | 'degraded' | 'unhealthy';
  version: string;
  uptime: number;
  timestamp: string;
  service?: string;
  environment?: string;
  checks?: Record<string, HealthCheck>;
  memory?: MemoryStats;
  cpu?: CpuStats;
  connections?: ConnectionStats;
  responseTimeMs: number;
}

export interface ServiceConfig {
  id: string;
  name: string;
  url: string;
  healthEndpoint: string;
  criticalService: boolean;
}

export interface DashboardState {
  services: Record<string, ServiceHealth | null>;
  lastUpdated: string;
  overallStatus: 'healthy' | 'degraded' | 'unhealthy';
}

export interface HistoricalDataPoint {
  timestamp: string;
  responseTimeMs: number;
  status: 'healthy' | 'degraded' | 'unhealthy';
  errorRate?: number;
}

export interface ServiceHistory {
  serviceId: string;
  dataPoints: HistoricalDataPoint[];
}
