import { usePanelEntity } from "@/hooks/usePanelEntity";
import type { PermissionAnswer } from "@/models/Permission";
import { permissionLabel } from "@/models/PermissionPanel";

interface RequiresLineProps {
  answer: PermissionAnswer | undefined;
}

// What keeps an answer from taking effect, with the sentence that decided it.
export const RequiresLine = ({ answer }: RequiresLineProps) => {
  const entity = usePanelEntity();
  const requires = answer?.requires;
  return (
    requires && (
      <p className="mt-1 text-xs text-pretty text-muted-foreground">
        Can't run while {permissionLabel(requires.action, undefined, entity.type)} is denied · {requires.sentence}
      </p>
    )
  );
};
