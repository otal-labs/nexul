import { useState } from "react";
import { Check, Copy } from "lucide-react";

import { SettingsCard } from "@/components/settings/SettingsCard";
import { Button } from "@/components/ui/button";
import { useCopyConnectionToken } from "@/hooks/AuthHooks";

export const ConnectDesktopCard = () => {
  const copyToken = useCopyConnectionToken();
  const [copied, setCopied] = useState(false);

  const copy = () =>
    copyToken.mutate(undefined, {
      onSuccess: () => {
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
      },
    });

  return (
    <SettingsCard
      id="connect-desktop"
      title="Connect the desktop app"
      description="Paste the token into the desktop app, then sign in there."
    >
      <div className="space-y-3 text-sm">
        <p className="text-muted-foreground">It holds this server's address, not your account.</p>
        <Button variant="outline" size="sm" onClick={copy} loading={copyToken.isPending} aria-live="polite">
          {copied && (
            <Check
              className="size-4 animate-in fade-in-0 zoom-in-50 blur-in-[2px] text-success duration-200 ease-out"
              aria-hidden
            />
          )}
          {!copied && <Copy className="size-4 animate-in fade-in-0 duration-150 ease-out" aria-hidden />}
          {copied ? "Copied" : "Copy connection token"}
        </Button>
      </div>
    </SettingsCard>
  );
};
