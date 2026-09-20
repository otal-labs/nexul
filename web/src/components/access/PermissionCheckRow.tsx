import type { ReactNode } from "react";

import { Checkbox } from "@/components/ui/checkbox";
import { cn } from "@/lib/utils";

interface PermissionCheckRowProps {
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  children: ReactNode;
  className?: string;
}

// PermissionsDialog's pickers use plain local state, not react-hook-form, so FormInput/FormSwitch don't apply.
export const PermissionCheckRow = ({
  checked,
  onCheckedChange,
  children,
  className,
}: PermissionCheckRowProps) => (
  <label
    className={cn(
      "flex cursor-pointer items-center gap-2 rounded px-2 py-1 text-sm hover:bg-accent",
      className,
    )}
  >
    <Checkbox checked={checked} onCheckedChange={onCheckedChange} />
    {children}
  </label>
);
