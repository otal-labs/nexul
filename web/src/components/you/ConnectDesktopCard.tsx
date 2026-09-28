import { useState } from "react";
import { Check, Copy } from "lucide-react";

import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { Button } from "@/components/ui/button";
import type { ConnectionToken } from "@/models/User";

export const ConnectDesktopCard = () => {
  const [copied, setCopied] = useState(false);
  const [pending, setPending] = useState(false);

  const copy = async () => {
    setPending(true);
    try {
      const { token } = (await api.post<ConnectionToken>("/api/auth/connection-token")).data;
      await navigator.clipboard.writeText(token);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setPending(false);
    }
  };

  return (
    <SettingsCard
      id="connect-desktop"
      title="Connect the desktop app"
      description="Paste this instance's connection token into the Nexul desktop app, then sign in there."
    >
      <div className="space-y-3 text-sm">
        <p className="text-muted-foreground">Holds only this server's address, never your account.</p>
        <Button variant="outline" size="sm" onClick={copy} disabled={pending} aria-live="polite">
          {copied && (
            <Check
              key="check"
              className="size-4 animate-in fade-in-0 zoom-in-50 blur-in-[2px] text-success duration-200 ease-out"
              aria-hidden
            />
          )}
          {!copied && <Copy key="copy" className="size-4 animate-in fade-in-0 duration-150 ease-out" aria-hidden />}
          {copied ? "Copied" : "Copy connection token"}
        </Button>
      </div>
    </SettingsCard>
  );
};
