import type { AutomationRunOutcome } from "@/enums/Automation";

// Mirrors internal/automations/runs.go's Run: one delivery attempt's execution report and logs.
export interface AutomationRun {
  id: string;
  automation_id: string;
  event_topic: string;
  event_id: string;
  outcome: AutomationRunOutcome;
  error?: string;
  started_at: string;
  finished_at: string;
  duration_ms: number;
  logs: string;
  created_at: string;
}
