import { HashIcon, LockIcon } from "lucide-react";
import { useId } from "react";

import { Switch } from "@/components/ui/switch";
import { cn } from "@/lib/utils";

interface PrivateChannelRowProps {
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  // A Restricted member's channel is private and stays so.
  fixed?: boolean;
  className?: string;
}

export const PrivateChannelRow = ({ checked, onCheckedChange, fixed = false, className }: PrivateChannelRowProps) => {
  const labelId = useId();
  const Icon = checked ? LockIcon : HashIcon;
  return (
    <div className={cn("flex items-center gap-3 py-3", className)}>
      <Icon className="size-4 shrink-0 text-muted-foreground" aria-hidden />
      <div className="min-w-0 flex-1 space-y-0.5">
        <p id={labelId} className="text-sm font-medium">
          Private channel
        </p>
        <p className="text-sm text-muted-foreground">
          {checked ? "Only members see it and read it." : "Everyone in the workspace sees it and reads it."}
        </p>
      </div>
      <Switch aria-labelledby={labelId} checked={checked} disabled={fixed} onCheckedChange={onCheckedChange} />
    </div>
  );
};
