import { RefreshCw } from "lucide-react";

import { QrFrame } from "@/components/you/QrFrame";
import { Button } from "@/components/ui/button";
import { useNow } from "@/hooks/useNow";
import type { ConnectCode } from "@/models/User";

// The link the system camera hands to the app; the in-app scanner parses the same thing.
const connectLink = (code: ConnectCode) => `nexul://connect?host=${encodeURIComponent(code.host)}&code=${code.code}`;

const formatRemaining = (ms: number) => {
  const seconds = Math.max(0, Math.ceil(ms / 1000));
  return `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, "0")}`;
};

interface IssuedConnectCodeProps {
  code: ConnectCode;
  onNewCode: () => void;
  generating: boolean;
}

// Mounts when a code arrives, so the countdown clock starts at that moment and not when the page opened.
export const IssuedConnectCode = ({ code, onNewCode, generating }: IssuedConnectCodeProps) => {
  const now = useNow(true);
  const remaining = new Date(code.expires_at).getTime() - now;
  const expired = remaining <= 0;

  return (
    <QrFrame
      link={connectLink(code)}
      dimmed={expired}
      action={
        expired && (
          <Button
            size="sm"
            className="absolute inset-0 m-auto w-fit animate-in fade-in-0 zoom-in-[0.97] duration-150 ease-out"
            onClick={onNewCode}
            loading={generating}
          >
            <RefreshCw className="size-4" aria-hidden />
            New code
          </Button>
        )
      }
    >
      <p className="text-muted-foreground">Works once, for two minutes. Nobody else can use it after your phone does.</p>
      <p className="font-mono text-xs tabular-nums">
        <span className="text-muted-foreground">Code </span>
        {code.code}
      </p>
      <p className="font-mono text-xs tabular-nums text-muted-foreground">{expired ? "Expired" : `Expires in ${formatRemaining(remaining)}`}</p>
    </QrFrame>
  );
};
