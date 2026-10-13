import { UsersIcon } from "lucide-react";

import { PersonAvatar } from "@/components/PersonAvatar";
import type { PanelTarget } from "@/hooks/usePanelTargets";
import { cn } from "@/lib/utils";

interface TargetMarkProps {
  target: PanelTarget;
  className?: string;
}

// A role is a muted people icon on a surface-2 tile; a person is their avatar.
export const TargetMark = ({ target, className }: TargetMarkProps) => (
  <>
    {target.kind === "role" && (
      <span className={cn("flex size-6 shrink-0 items-center justify-center rounded-md bg-surface-2 text-muted-foreground ring-1 ring-border ring-inset", className)}>
        <UsersIcon className="size-[55%]" aria-hidden />
      </span>
    )}
    {target.kind === "person" && <PersonAvatar login={target.login} src={target.avatarUrl} className={cn("size-6", className)} label={target.name} />}
  </>
);
