import { HashIcon, LockIcon } from "lucide-react";

import { ActionSwitch } from "@/components/action/ActionSwitch";
import { cn } from "@/lib/utils";

interface PrivateChannelRowProps {
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  className?: string;
}

export const PrivateChannelRow = ({ checked, onCheckedChange, className }: PrivateChannelRowProps) => {
  const Icon = checked ? LockIcon : HashIcon;
  return (
    <div className={cn("flex items-center gap-3 py-3", className)}>
      <Icon className="size-4 shrink-0 text-muted-foreground" aria-hidden />
      <div className="min-w-0 flex-1 space-y-0.5">
        <p className="text-sm font-medium">
          Private channel
        </p>
        <p className="text-sm text-muted-foreground">
          {checked ? "Only its members can see it." : "Everyone in the workspace can read it."}
        </p>
      </div>
      <ActionSwitch permission="channels:write" verb="change who sees channels" aria-label="Private channel" checked={checked} onCheckedChange={onCheckedChange} />
    </div>
  );
};
