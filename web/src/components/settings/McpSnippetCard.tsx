import { useState } from "react";
import { Check, Copy } from "lucide-react";

import { Button } from "@/components/ui/button";

interface McpSnippetCardProps {
  label: string;
  filename: string;
  snippet: string;
}

// Dumb copy-paste block — the caller decides what JSON goes in the snippet.
export const McpSnippetCard = ({ label, filename, snippet }: McpSnippetCardProps) => {
  const [copied, setCopied] = useState(false);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(snippet);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Clipboard may be unavailable; the block stays visible for manual copy.
    }
  };

  return (
    <div className="space-y-2 rounded-md border border-border p-3">
      <div className="flex items-center justify-between gap-2">
        <p className="text-sm font-medium">
          {label} <span className="text-xs text-muted-foreground">· {filename}</span>
        </p>
        <Button type="button" variant="outline" size="sm" onClick={copy}>
          {copied && <Check className="size-4" />}
          {!copied && <Copy className="size-4" />}
          {copied ? "Copied" : "Copy"}
        </Button>
      </div>
      <pre className="overflow-x-auto rounded bg-muted p-2 font-mono text-xs">{snippet}</pre>
    </div>
  );
};
