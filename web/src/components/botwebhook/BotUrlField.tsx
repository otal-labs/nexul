import { useState } from "react";
import { Check, Copy } from "lucide-react";

interface BotUrlFieldProps {
  name: string;
  url: string;
}

// The URL in a read-only field with the Copy button joined to its right edge.
export const BotUrlField = ({ name, url }: BotUrlFieldProps) => {
  const [copied, setCopied] = useState(false);
  const copy = async () => {
    await navigator.clipboard.writeText(url);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="flex min-w-0 overflow-hidden rounded-md border border-input bg-background focus-within:ring-2 focus-within:ring-ring">
      <input
        readOnly
        value={url}
        aria-label={`${name}'s webhook URL`}
        onFocus={(e) => e.currentTarget.select()}
        className="min-w-0 flex-1 bg-transparent px-2.5 py-1.5 font-mono text-xs text-muted-foreground outline-none"
      />
      <button
        type="button"
        onClick={() => void copy()}
        className="flex shrink-0 items-center gap-1.5 border-l border-input px-3 text-xs font-medium transition-colors duration-150 ease-standard hover:bg-accent focus-visible:bg-accent focus-visible:outline-none"
      >
        {copied && <Check className="size-3.5" aria-hidden />}
        {!copied && <Copy className="size-3.5" aria-hidden />}
        {copied ? "Copied" : "Copy"}
      </button>
    </div>
  );
};
