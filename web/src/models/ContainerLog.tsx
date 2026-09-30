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
