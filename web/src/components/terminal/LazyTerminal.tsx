import { lazy } from "react";

// The terminal's xterm loads only once one opens.
export const LazyTerminalDialog = lazy(() =>
  import("@/components/terminal/TerminalDialog").then((m) => ({
    default: m.TerminalDialog,
  })),
);
export const LazyTerminalTranscript = lazy(() =>
  import("@/components/terminal/TerminalTranscript").then((m) => ({
    default: m.TerminalTranscript,
  })),
);
