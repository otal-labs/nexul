import { useState } from "react";
import { CloudIcon, RouteIcon, Trash2 } from "lucide-react";

import { ConfirmDestroyButton } from "@/components/settings/ConfirmDestroyButton";
import { NoFillBadge } from "@/components/ui/badge";
import { useDeleteExposure } from "@/hooks/DnsHooks";
import { leavingRowClass } from "@/hooks/useRowGlide";
import { cn } from "@/lib/utils";
import type { Exposure, Gateway } from "@/models/DNS";
import type { Container } from "@/models/Stack";

interface ExposureRowProps {
  // Called as the row starts to leave, so the list can glide the rows under it up once it is gone.
  onLeave?: () => void;
  exposure: Exposure;
  gateway: Gateway | undefined;
  container: Container | undefined;
}

// Each row owns its removal, so unexposing one hostname spins only its own button.
export const ExposureRow = ({ onLeave, exposure, gateway, container }: ExposureRowProps) => {
  const removeExposure = useDeleteExposure();
  const [leaving, setLeaving] = useState(false);
  return (
    <li data-leaving={leaving || undefined} className={cn(leavingRowClass, "flex flex-wrap items-center gap-x-3 gap-y-1 px-3 py-2.5")}>
      <div className="min-w-0 flex-1 space-y-0.5">
        <a
          href={`https://${exposure.hostname}`}
          target="_blank"
          rel="noreferrer"
          className="wrap-anywhere font-mono text-sm underline decoration-border underline-offset-2 transition-colors duration-150 ease-standard hover:decoration-foreground"
        >
          {exposure.hostname}
        </a>
        <p className="flex flex-wrap items-center gap-x-2 font-mono text-xs text-muted-foreground">
          <span>
            → {container?.name ?? exposure.service}:{exposure.port}
          </span>
          {gateway && (
            <NoFillBadge icon={gateway.kind === "tunnel" ? CloudIcon : RouteIcon} color="text-muted-foreground">
              {gateway.kind}
            </NoFillBadge>
          )}
        </p>
      </div>
      <ConfirmDestroyButton
        icon={Trash2}
        idleLabel={`Unexpose ${exposure.hostname}`}
        confirmLabel="Unexpose"
        loading={removeExposure.isPending}
        onConfirm={() => {
          setLeaving(true);
          onLeave?.();
          removeExposure.mutate(exposure.id, { onError: () => setLeaving(false) });
        }}
      />
    </li>
  );
};
