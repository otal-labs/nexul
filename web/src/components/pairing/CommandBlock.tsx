import { Check, Copy } from "lucide-react";
import { Fragment, useState } from "react";

interface CommandBlockProps {
  // The shell the lines run in, shown as the window title.
  shell: string;
  lines: string[];
  prompt?: string;
}

// A static terminal window; a line wraps between its arguments, and inside one only when it is wider than the window.
export const CommandBlock = ({ shell, lines, prompt = "$" }: CommandBlockProps) => {
  const [copied, setCopied] = useState(false);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(lines.join("\n"));
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Clipboard may be unavailable; the lines stay selectable for a manual copy.
    }
  };

  return (
    <div className="terminal-window min-w-0 overflow-hidden">
      <div className="terminal-window__bar">
        <span className="terminal-window__dot" aria-hidden />
        <span className="terminal-window__dot" aria-hidden />
        <span className="terminal-window__dot" aria-hidden />
        <span className="terminal-window__title flex-1">{shell}</span>
        <button
          type="button"
          onClick={copy}
          aria-label={copied ? "Copied" : `Copy ${shell} commands`}
          className="terminal-window__action"
        >
          {copied && <Check className="size-3.5" aria-hidden />}
          {!copied && <Copy className="size-3.5" aria-hidden />}
        </button>
      </div>
      <pre className="terminal-window__body !overflow-x-visible text-xs leading-relaxed">
        {lines.map((line, index) => (
          <code key={index} className="flex gap-2">
            <span className="terminal-window__prompt shrink-0" aria-hidden>
              {prompt}
            </span>
            <span className="min-w-0 whitespace-pre-wrap">
              {(line.match(/(?:"[^"]*"|\S)+/g) ?? []).map((arg, i) => (
                <Fragment key={i}>
                  {i > 0 && " "}
                  <span className="inline-block max-w-full wrap-anywhere">{arg}</span>
                </Fragment>
              ))}
            </span>
          </code>
        ))}
      </pre>
    </div>
  );
};
