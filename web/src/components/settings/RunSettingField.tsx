import type { ReactNode } from "react";

interface RunSettingFieldProps {
  label: string;
  children: ReactNode;
}

// One setting in a Projects row's editor: a muted label beside its control, under it in a narrow well.
export const RunSettingField = ({ label, children }: RunSettingFieldProps) => (
  <div className="grid gap-1.5 py-2 @[22rem]:grid-cols-[8rem_minmax(0,1fr)] @[22rem]:items-start @[22rem]:gap-4">
    <p className="text-sm text-muted-foreground @[22rem]:pt-2">{label}</p>
    <div className="grid min-w-0">{children}</div>
  </div>
);
