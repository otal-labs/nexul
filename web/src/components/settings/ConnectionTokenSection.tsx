import { useState } from "react";
import { Check, Copy, KeyRound } from "lucide-react";

import { SettingsCard } from "@/components/settings/SettingsCard";
import { Button } from "@/components/ui/button";
import { useGenerateConnectionToken } from "@/hooks/AuthHooks";

export const ConnectionTokenSection = () => {
  const generateToken = useGenerateConnectionToken();
  const [showToken, setShowToken] = useState(false);
  const [copied, setCopied] = useState(false);

  const copyToken = async () => {
    if (!generateToken.data) return;
    try {
      await navigator.clipboard.writeText(generateToken.data.token);
      setCopied(true);
    } catch {
      // Clipboard may be unavailable; the box stays visible for manual copy.
    }
  };

  return (
    <SettingsCard
      id="tokens"
      title="Connection token"
      description="Import this token into a standalone client (e.g. the desktop app) so it knows where to
        connect. It carries server information only — not a credential."
    >
      <Button
        type="button"
        variant="outline"
        onClick={() => {
          setShowToken(false);
          setCopied(false);
          generateToken.mutate(undefined, { onSuccess: () => setShowToken(true) });
        }}
        disabled={generateToken.isPending}
      >
        <KeyRound className="size-4" />
        {generateToken.isPending ? "Generating…" : "Generate connection token"}
      </Button>
      {showToken && generateToken.data && (
        <div className="animate-in fade-in-0 slide-in-from-top-1 mt-4 space-y-2 rounded-md bg-muted p-4 duration-200 ease-out">
          <p className="break-all font-mono text-xs">{generateToken.data.token}</p>
          <p className="text-xs text-muted-foreground">
            Instance: {generateToken.data.instance_url} · Expires{" "}
            {new Date(generateToken.data.expires_at).toLocaleString()}
          </p>
          <Button type="button" variant="outline" size="sm" onClick={copyToken}>
            {copied && <Check className="size-4" />}
            {!copied && <Copy className="size-4" />}
            {copied ? "Copied" : "Copy"}
          </Button>
        </div>
      )}
    </SettingsCard>
  );
};
