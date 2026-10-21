import { CopyButton } from "@/components/settings/CopyButton";

interface AutomationTokenRevealProps {
  token: string;
}

// The raw token shows exactly once; a host runs the automation on credentials of its own.
export const AutomationTokenReveal = ({ token }: AutomationTokenRevealProps) => (
  <div className="animate-in fade-in-0 slide-in-from-top-1 space-y-2 rounded-md bg-muted p-4 duration-200 ease-out">
    <p className="text-sm font-medium">Copy this token now. It won&apos;t be shown again.</p>
    <p className="text-sm text-muted-foreground">An automations host runs the automation without it.</p>
    <div className="flex items-center gap-2">
      <p className="min-w-0 flex-1 truncate rounded-md bg-card p-2 font-mono text-xs">{token}</p>
      <CopyButton value={token} label="Copy" />
    </div>
  </div>
);
