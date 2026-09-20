import { NoDataDisplay } from "@/components/NoDataDisplay";

interface AutomationRunLogViewProps {
  logs: string;
}

// Captured run logs (capped at 1MB server-side) framed as a
// terminal window, scrollable so a long run never blows out the sheet.
export const AutomationRunLogView = ({ logs }: AutomationRunLogViewProps) => (
  <div className="terminal-window">
    <div className="terminal-window__bar">
      <span className="terminal-window__dot" aria-hidden />
      <span className="terminal-window__dot" aria-hidden />
      <span className="terminal-window__dot" aria-hidden />
      <span className="terminal-window__title">Logs</span>
    </div>
    {logs.trim() === "" && (
      <div className="terminal-window__body">
        <NoDataDisplay message="No logs captured" size="compact" />
      </div>
    )}
    {logs.trim() !== "" && (
      <pre className="terminal-window__body max-h-96 overflow-y-auto whitespace-pre-wrap break-words">{logs}</pre>
    )}
  </div>
);
