import { useState } from "react";
import { RefreshCw } from "lucide-react";

import { cn } from "@/lib/utils";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { Button } from "@/components/ui/button";
import { useNow } from "@/hooks/useNow";
import { PhoneConnected } from "@/components/you/PhoneConnected";
import { usePrototypeStore } from "@/components/you/prototypeStore";

const CODE_TTL_MS = 2 * 60_000;

const formatRemaining = (ms: number) => {
  const seconds = Math.max(0, Math.ceil(ms / 1000));
  return `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, "0")}`;
};

export const ConnectPhoneCard = () => {
  const [issuedAt, setIssuedAt] = useState(() => Date.now());
  const now = useNow(true);
  const remaining = issuedAt + CODE_TTL_MS - now;
  const expired = remaining <= 0;
  const connected = usePrototypeStore((s) => (s.variant === "toast" ? null : s.connected));

  return (
    <SettingsCard id="connect-phone" title="Connect a phone" description="Open the Nexul app on your phone and scan this code.">
      {connected && <PhoneConnected phone={connected} />}
      {!connected && (
      <div className="flex items-start gap-5">
        <div className="relative shrink-0 rounded-md bg-white p-2.5">
          <img
            key={issuedAt}
            src="/proto-connect-qr.svg"
            alt="Sign-in code for the Nexul app"
            className={cn("size-32 animate-in fade-in-0 duration-150 ease-out transition-opacity ease-standard", expired && "opacity-10")}
          />
          {expired && (
            <Button size="sm" className="absolute inset-0 m-auto w-fit animate-in fade-in-0 zoom-in-[0.97] duration-150 ease-out" onClick={() => setIssuedAt(Date.now())}>
              <RefreshCw className="size-4" aria-hidden />
              New code
            </Button>
          )}
        </div>
        <div className="min-w-0 space-y-3 text-sm">
          <p className="text-muted-foreground">Works once, for two minutes. Nobody else can use it after your phone does.</p>
          <p className="font-mono text-xs tabular-nums">
            <span className="text-muted-foreground">Code </span>NXC-7Q4F-K92M
          </p>
          <p className="font-mono text-xs tabular-nums text-muted-foreground">
            {expired ? "Expired" : `Expires in ${formatRemaining(remaining)}`}
          </p>
          <Button variant="outline" size="sm" onClick={() => setIssuedAt(Date.now())}>
            <RefreshCw className="size-4" aria-hidden />
            New code
          </Button>
        </div>
      </div>
      )}
    </SettingsCard>
  );
};
