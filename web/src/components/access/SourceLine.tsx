import { Button } from "@/components/ui/button";
import { useCanManageAccess } from "@/hooks/useCanManageAccess";
import { usePanelEntity } from "@/hooks/usePanelEntity";
import { usePanelTargets } from "@/hooks/usePanelTargets";
import { usePermissionsDialog } from "@/hooks/usePermissionsDialog";
import { usePermissionPanelStore } from "@/stores/permissionPanelStore";
import type { PermissionSource, PermissionVerdict } from "@/models/Permission";
import { sentenceParts, targetKey, targetScope, targetsProject, verdictTone, type SentencePart } from "@/models/PermissionPanel";
import { cn } from "@/lib/utils";

const dotClass = {
  allowed: "bg-success",
  denied: "bg-destructive",
  none: "bg-muted-foreground/50",
};

const nameClass = "font-medium text-foreground";
const linkClass = "h-auto p-0 align-baseline text-[length:inherit] font-medium text-foreground underline decoration-border underline-offset-2 hover:decoration-foreground";

// Another project this sentence names opens its own panel, for whoever manages that project's access.
const EntityName = ({ text, source }: { text: string; source: PermissionSource }) => {
  const entity = usePanelEntity();
  const other = source.entity_type === "project" && source.entity_id !== entity.id;
  const canManage = useCanManageAccess(other ? { type: "project", id: source.entity_id } : undefined);
  const openPanel = usePermissionsDialog();
  const project = { type: "project" as const, id: source.entity_id, name: text, workspaceId: entity.workspaceId, projectId: source.entity_id, folderId: "" };
  return (
    <>
      {!(other && canManage) && <span className={nameClass}>{text}</span>}
      {other && canManage && (
        <Button type="button" variant="link" className={linkClass} onClick={() => void openPanel(project)}>
          {text}
        </Button>
      )}
    </>
  );
};

// A role this sentence names selects it in the column, unless the doc's or folder's project doesn't list it.
const RoleName = ({ text, source }: { text: string; source: PermissionSource }) => {
  const entity = usePanelEntity();
  const add = usePermissionPanelStore((s) => s.add);
  const { targetOf, blockedFor } = usePanelTargets(targetScope(entity, targetsProject(entity)));
  const key = targetKey("role", source.target_id);
  const target = targetOf(key);
  const opens = target.known && !blockedFor(target);
  return (
    <>
      {!opens && <span className={nameClass}>{text}</span>}
      {opens && (
        <Button type="button" variant="link" className={linkClass} onClick={() => add(key)}>
          {text}
        </Button>
      )}
    </>
  );
};

const Part = ({ part, source }: { part: SentencePart; source: PermissionSource }) => (
  <>
    {!part.names && <span>{part.text}</span>}
    {part.names === "entity" && <EntityName text={part.text} source={source} />}
    {part.names === "role" && <RoleName text={part.text} source={source} />}
  </>
);

interface SourceLineProps {
  verdict: PermissionVerdict;
  className?: string;
}

// The dot is the answer; the sentence says what decided it, the entity and role it names as links to their panels.
export const SourceLine = ({ verdict, className }: SourceLineProps) => (
  <p className={cn("flex min-w-0 items-start gap-2 text-xs text-muted-foreground", className)}>
    <span className={cn("mt-[5px] size-1.5 shrink-0 rounded-full", dotClass[verdictTone(verdict.allowed, verdict.source)])} aria-hidden />
    <span className="min-w-0 text-pretty">
      {sentenceParts(verdict.source).map((part, i) => (
        <Part key={i} part={part} source={verdict.source} />
      ))}
    </span>
  </p>
);
