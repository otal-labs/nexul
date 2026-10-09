import { CopyButton } from "@/components/settings/CopyButton";
import { useFetchSettings } from "@/hooks/AuthHooks";

interface AutomationTokenRevealProps {
  token: string;
}

// The raw token shows exactly once, plus SDK commands pre-filled with the token and instance URL.
export const AutomationTokenReveal = ({ token }: AutomationTokenRevealProps) => {
  const { data: settings } = useFetchSettings();
  const instanceUrl = settings?.instance_url || "https://your-instance.example.com";
  const initCommand = `npx @nexul/sdk init --url ${instanceUrl} --token ${token}`;

  return (
    <div className="animate-in fade-in-0 slide-in-from-top-1 space-y-3 rounded-md bg-muted p-4 duration-200 ease-out">
      <div className="space-y-2">
        <p className="text-sm font-medium">Copy this token now. It won&apos;t be shown again.</p>
        <div className="flex items-center gap-2">
          <p className="min-w-0 flex-1 truncate rounded-md bg-card p-2 font-mono text-xs">{token}</p>
          <CopyButton value={token} label="Copy" />
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
