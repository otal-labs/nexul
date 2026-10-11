export type LogStream = "stdout" | "stderr";

export type LogFilter = "all" | "errors";

// One message from the logs socket is a batch: {lines: [{ts, stream, line}]}.
export interface ContainerLogWireLine {
  ts: string;
  stream: LogStream;
  line: string;
}

export interface ContainerLogLine {
  seq: number;
  ts: string;
  stream: LogStream;
  text: string;
}

// forbidden: the server refused this viewer, so retrying is pointless. ended: the container stopped writing.
export type LogStatus = "connecting" | "live" | "offline" | "ended" | "forbidden";

export interface ContainerLogs {
  lines: ContainerLogLine[];
  status: LogStatus;
  reason: string | undefined;
}

// The most lines a view keeps in memory; older ones fall off the top.
export const MAX_LOG_LINES = 5000;

// Where one container's logs are read: path under /ws for the live tail and under /api for the snapshot.
export interface ContainerLogSource {
  path: string;
  name: string;
  forbidden: string;
}

export const stackLogSource = (stackId: string, service: string): ContainerLogSource => ({
  path: `/stacks/${stackId}/services/${encodeURIComponent(service)}/logs`,
  name: service,
  forbidden: "You can't read logs for this stack.",
});

export const computerLogSource = (computerId: string, container: string): ContainerLogSource => ({
  path: `/pairing/computers/${computerId}/containers/${encodeURIComponent(container)}/logs`,
  name: container,
  forbidden: "This container's computer isn't one of yours.",
});
