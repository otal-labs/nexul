import type { ContainerLogLine } from "@/models/Stack";

export const buildLogsURL = (host: string, token: string, stackId: string, service: string, tail: number): string =>
  `${host.replace(/^http/i, "ws")}/ws/stacks/${stackId}/services/${encodeURIComponent(service)}/logs?tail=${tail}&token=${encodeURIComponent(token)}`;

// Every socket message is one batch: {lines: [{ts, stream, line}]}.
export const parseLogFrame = (raw: string): ContainerLogLine[] => {
  const data = JSON.parse(raw) as { lines?: unknown };
  if (!Array.isArray(data.lines)) throw new Error("frame has no lines");
  return data.lines as ContainerLogLine[];
};
