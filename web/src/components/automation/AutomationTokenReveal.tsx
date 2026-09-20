import { useState } from "react";
import { Check, Copy } from "lucide-react";

import { Button } from "@/components/ui/button";
import { useFetchSettings } from "@/hooks/AuthHooks";

interface AutomationTokenRevealProps {
  token: string;
}

// The raw token shows exactly once, plus SDK commands pre-filled with the token and instance URL.
export const AutomationTokenReveal = ({ token }: AutomationTokenRevealProps) => {
  const { data: settings } = useFetchSettings();
  const [copied, setCopied] = useState(false);
  const instanceUrl = settings?.instance_url || "https://your-instance.example.com";
  const initCommand = `npx @nexul/sdk init --url ${instanceUrl} --token ${token}`;

  const copyToken = async () => {
    try {
      await navigator.clipboard.writeText(token);
      setCopied(true);
    } catch {
      // Clipboard may be unavailable; the box stays visible for manual copy.
    }
  };

  return (
    <div className="animate-in fade-in-0 slide-in-from-top-1 space-y-3 rounded-md bg-muted p-4 duration-200 ease-out">
      <div className="space-y-2">
        <p className="text-sm font-medium">Copy this token now — it won&apos;t be shown again.</p>
        <div className="flex items-center gap-2">
          <p className="min-w-0 flex-1 truncate rounded bg-card p-2 font-mono text-xs">{token}</p>
          <Button type="button" variant="outline" size="sm" onClick={copyToken}>
            {copied && <Check className="size-4" />}
            {!copied && <Copy className="size-4" />}
            {copied ? "Copied" : "Copy"}
          </Button>
        </div>
      </div>
      <div className="terminal-window">
        <div className="terminal-window__bar">
          <span className="terminal-window__dot" aria-hidden />
          <span className="terminal-window__dot" aria-hidden />
          <span className="terminal-window__dot" aria-hidden />
          <span className="terminal-window__title">Set up the SDK</span>
        </div>
        <pre className="terminal-window__body whitespace-pre-wrap break-all">
          {initCommand}
          {"\n"}npx @nexul/sdk push
        </pre>
      </div>
    </div>
  );
};
