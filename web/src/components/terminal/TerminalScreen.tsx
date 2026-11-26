import { useEffect, useRef } from "react";

import { buildSocketURL } from "@/lib/live";
import { connectTerminal, mountTerminal, type TerminalState } from "@/lib/terminal";
import { useLatestCallback } from "@/hooks/useLatestCallback";
import { useSessionStore } from "@/stores/sessionStore";

interface TerminalScreenProps {
  // The terminal's socket, for a live one; without it the screen replays transcript and takes no keys.
  socketPath?: string;
  transcript?: string;
  onState?: (state: TerminalState) => void;
}

// One xterm on the console fill: live over its socket, or a replay of what a closed terminal printed.
export const TerminalScreen = ({ socketPath, transcript, onState }: TerminalScreenProps) => {
  const host = useRef<HTMLDivElement>(null);
  const report = useLatestCallback((state: TerminalState) => onState?.(state));

  useEffect(() => {
    if (!host.current) return;
    const { term, dispose } = mountTerminal(host.current, !socketPath);
    if (transcript) term.write(transcript);
    let disconnect = () => {};
    // A tick late, so StrictMode's throwaway mount never takes the one socket a terminal accepts.
    const timer = setTimeout(() => {
      const token = useSessionStore.getState().token;
      if (socketPath && token) disconnect = connectTerminal(term, buildSocketURL(token, socketPath), report);
    });
    return () => {
      clearTimeout(timer);
      disconnect();
      dispose();
    };
  }, [socketPath, transcript, report]);

  return <div ref={host} className="h-[60vh] min-h-80 overflow-hidden rounded-lg border border-console-edge bg-console p-2" />;
};
