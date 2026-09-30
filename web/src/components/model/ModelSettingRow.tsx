import type { ReactNode } from "react";

interface ModelSettingRowProps {
  label: string;
  description: string;
  children: ReactNode;
}

// A settings row: the setting's name and what it does on the left, the model and its options on the right.
export const ModelSettingRow = ({ label, description, children }: ModelSettingRowProps) => (
  <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between md:gap-6">
    <div className="min-w-0">
      <p className="text-sm font-medium">{label}</p>
      <p className="text-xs text-muted-foreground">{description}</p>
    </div>
    <div className="flex min-w-0 flex-wrap items-center gap-2 md:justify-end">{children}</div>
  </div>
);
