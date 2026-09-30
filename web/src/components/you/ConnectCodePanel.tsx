import { useMemo } from "react";
import { RefreshCw } from "lucide-react";
import { renderSVG } from "uqr";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Button } from "@/components/ui/button";
import { useConnectCode } from "@/hooks/AuthHooks";
import { useNow } from "@/hooks/useNow";
import type { ConnectCode } from "@/models/User";
import { cn } from "@/lib/utils";

// The link the system camera hands to the app; the in-app scanner parses the same thing.
const connectLink = (code: ConnectCode) => `nexul://connect?host=${encodeURIComponent(code.host)}&code=${code.code}`;

// Generated locally from our own string, so a data URL carries it without touching innerHTML.
const qrDataUrl = (link: string) =>
  `data:image/svg+xml,${encodeURIComponent(renderSVG(link, { border: 0, whiteColor: "#ffffff", blackColor: "#000000" }))}`;

const formatRemaining = (ms: number) => {
  const seconds = Math.max(0, Math.ceil(ms / 1000));
  return `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, "0")}`;
};

export const ConnectCodePanel = () => {
  const { data, error, isPending, isFetching, refetch } = useConnectCode();
  const qr = useMemo(() => data && qrDataUrl(connectLink(data)), [data]);
  const now = useNow(!!data);
  const remaining = data ? new Date(data.expires_at).getTime() - now : 0;
  const expired = !data || remaining <= 0;

  return (
    <>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {(data || error) && (
        <div className="@container mt-3">
          {/* Two cards share the row from 1024px, so a card can be 240px wide: the text drops under the QR until the card has room. */}
          <div className="flex flex-col items-start gap-5 @sm:flex-row">
            <div className="relative size-[9.25rem] shrink-0 rounded-md bg-white p-2.5">
              {qr && (
                <img
                  key={data?.code}
                  src={qr}
                  alt="Sign-in code for the Nexul app"
                  className={cn(
                    "size-32 animate-in fade-in-0 animation-duration-150 ease-out transition-opacity duration-200",
                    expired && "opacity-10",
                  )}
                />
              )}
              {expired && (
                <Button
                  size="sm"
                  className="absolute inset-0 m-auto w-fit animate-in fade-in-0 zoom-in-[0.97] duration-150 ease-out"
                  onClick={() => void refetch()}
                  loading={isFetching}
                >
                  <RefreshCw className="size-4" aria-hidden />
                  New code
                </Button>
              )}
            </div>
            {data && (
              <div className="min-w-0 space-y-3 text-sm">
                <p className="text-muted-foreground">Works once, for two minutes. Nobody else can use it after your phone does.</p>
                <p className="font-mono text-xs tabular-nums">
                  <span className="text-muted-foreground">Code </span>
                  {data.code}
                </p>
                <p className="font-mono text-xs tabular-nums text-muted-foreground">
                  {expired ? "Expired" : `Expires in ${formatRemaining(remaining)}`}
                </p>
              </div>
            )}
          </div>
        </div>
      )}
    </>
  );
};
