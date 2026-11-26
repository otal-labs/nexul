import { FitAddon } from "@xterm/addon-fit";
import { Terminal } from "@xterm/xterm";
import "@xterm/xterm/css/xterm.css";

// Live while the socket is open; ended once the shell exited; lost when the socket closed without an exit.
export type TerminalStatus = "connecting" | "live" | "ended" | "lost";

export interface TerminalState {
  status: TerminalStatus;
  exit?: number;
}

const cssToken = (name: string) => getComputedStyle(document.documentElement).getPropertyValue(name).trim();

// An xterm filling host and following its size, in the console's colours; read-only takes no keys.
export const mountTerminal = (host: HTMLElement, readOnly: boolean) => {
  const ink = cssToken("--console-ink");
  const term = new Terminal({
    fontFamily: cssToken("--font-mono-stack"),
    fontSize: 13,
    lineHeight: 1.2,
    scrollback: 5000,
    cursorBlink: !readOnly,
    disableStdin: readOnly,
    theme: {
      background: cssToken("--console"),
      foreground: ink,
      cursor: ink,
      selectionBackground: `${ink}4d`,
    },
  });
  const fit = new FitAddon();
  term.loadAddon(fit);
  term.open(host);
  fit.fit();
  const observer = new ResizeObserver(() => fit.fit());
  observer.observe(host);
  return {
    term,
    dispose: () => {
      observer.disconnect();
      term.dispose();
    },
  };
};

// Carries term over the terminal socket at url: keys and sizes out, the shell's bytes in, its exit as a text frame.
// The returned function closes the socket, which ends the shell.
export const connectTerminal = (term: Terminal, url: string, onState: (state: TerminalState) => void) => {
  const ws = new WebSocket(url);
  ws.binaryType = "arraybuffer";
  const encoder = new TextEncoder();
  let exit: number | undefined;
  const send = (data: string | Uint8Array<ArrayBuffer>) => {
    if (ws.readyState === WebSocket.OPEN) ws.send(data);
  };
  const sendSize = () => send(JSON.stringify({ cols: term.cols, rows: term.rows }));
  ws.onopen = () => {
    onState({ status: "live" });
    sendSize();
    term.focus();
  };
  ws.onmessage = (ev: MessageEvent<ArrayBuffer | string>) => {
    if (typeof ev.data !== "string") {
      term.write(new Uint8Array(ev.data));
      return;
    }
    exit = (JSON.parse(ev.data) as { exit: number }).exit;
  };
  ws.onclose = () => {
    // Nothing reads keys any more, so the screen stops taking them and hides its cursor.
    term.options.disableStdin = true;
    term.write("\x1b[?25l");
    onState(exit === undefined ? { status: "lost" } : { status: "ended", exit });
  };
  const subscriptions = [
    term.onData((data) => send(encoder.encode(data))),
    term.onBinary((data) => send(Uint8Array.from(data, (c) => c.charCodeAt(0)))),
    term.onResize(sendSize),
  ];
  return () => {
    subscriptions.forEach((s) => s.dispose());
    ws.onclose = null;
    ws.close();
  };
};
