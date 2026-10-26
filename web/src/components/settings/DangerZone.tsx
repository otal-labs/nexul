import type { ReactNode } from "react";
import { TriangleAlert } from "lucide-react";

import { SettingsCard } from "@/components/settings/SettingsCard";
import { SettingsHint } from "@/components/settings/SettingsHint";

interface DangerZoneProps {
  children: ReactNode;
}

// The one danger zone: a destructive-edged card of action rows, each naming what it does and what is lost.
export const DangerZone = ({ children }: DangerZoneProps) => (
  <SettingsCard id="danger-zone" title="Danger zone" danger icon={TriangleAlert}>
    <ul className="-my-2 divide-y divide-border">{children}</ul>
  </SettingsCard>
);

interface DangerActionProps {
  title: string;
  consequence: ReactNode;
  // What goes with it, or what blocks it: hostnames, counts.
  details?: ReactNode;
  action: ReactNode;
}

export const DangerAction = ({ title, consequence, details, action }: DangerActionProps) => (
  <li className="@container py-3">
    <div className="flex flex-col gap-3 @sm:flex-row @sm:items-start @sm:justify-between @sm:gap-6">
      <div className="min-w-0 space-y-1">
        <p className="text-sm font-medium">{title}</p>
        <SettingsHint>{consequence}</SettingsHint>
        {details}
      </div>
      <div className="shrink-0">{action}</div>
    </div>
  </li>
);

export const dangerButtonClass = "border-destructive/40 text-destructive hover:border-destructive/60 hover:bg-destructive/10 hover:text-destructive";
