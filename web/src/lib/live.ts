import { resolveWSBase } from "@/api/client";

export const buildLiveURL = (token: string) => {
  const base = import.meta.env.VITE_WS_URL || `${resolveWSBase()}/ws/events`;
  return `${base}?token=${encodeURIComponent(token)}`;
};

// Same session-token query auth as the events socket; the tail is how many past lines arrive before the follow.
export const buildLogsURL = (token: string, stackId: string, service: string, tail: number) =>
  `${resolveWSBase()}/ws/stacks/${stackId}/services/${encodeURIComponent(service)}/logs?tail=${tail}&token=${encodeURIComponent(token)}`;
