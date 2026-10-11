import { personLabel, unknownPerson } from "@nexul/client-core/person";

import { Button } from "@/components/ui/button";
import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PanelPaneHeader } from "@/components/access/PanelPaneHeader";
import { PermissionGroupSection } from "@/components/access/PermissionGroupSection";
import { PermissionRuleRow } from "@/components/access/PermissionRuleRow";
import { useFetchExplanation, useFetchPermissionCatalog } from "@/hooks/PermissionHooks";
import { useFetchWorkspacePeople } from "@/hooks/PeopleHooks";
import { usePanelEntity } from "@/hooks/usePanelEntity";
import { usePermissionRules } from "@/hooks/usePermissionRules";
import { usePermissionPanelStore } from "@/stores/permissionPanelStore";
import type { PermissionExplanation, PermissionInfo } from "@/models/Permission";
import { groupActions, permissionLabel, type PanelGroup } from "@/models/PermissionPanel";

// The role behind a person's answers, read from the sources that name one.
const heldRole = (explanation: PermissionExplanation | undefined) =>
  explanation?.effective.map((a) => a.fallback.source).find((s) => s.target_type === "role" && (s.kind === "role" || s.kind === "owner"))?.target_name;

interface RuleGroupProps {
  userId: string;
  group: PanelGroup;
  explanation: PermissionExplanation;
  catalog: PermissionInfo[] | undefined;
}

const RuleGroup = ({ userId, group, explanation, catalog }: RuleGroupProps) => (
  <PermissionGroupSection name={group.name}>
    {group.actions.map((action) => (
      <PermissionRuleRow
        key={action}
        userId={userId}
        action={action}
        label={permissionLabel(action, catalog)}
        answer={explanation.effective.find((a) => a.action === action)}
      />
    ))}
  </PermissionGroupSection>
);

const RulesFor = ({ userId }: { userId: string }) => {
  const entity = usePanelEntity();
  const { data: people } = useFetchWorkspacePeople(entity.workspaceId);
  const { data: catalog } = useFetchPermissionCatalog();
  const { data: explanation, error, isPending } = useFetchExplanation(entity.type, entity.id, userId);
  const { actions, clear } = usePermissionRules();
  const drop = usePermissionPanelStore((s) => s.drop);
  const person = people?.find((p) => p.user_id === userId) ?? unknownPerson(userId);
  const role = heldRole(explanation);

  return (
    <div className="min-w-0 flex-1 space-y-5">
      <PanelPaneHeader
        person={person}
        title={personLabel(person)}
        line={role ? `Person · holds ${role}` : "Person"}
        action={
          <Button
            variant="ghost"
            size="sm"
            className="text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
            onClick={() => {
              clear(userId);
              drop(userId);
            }}
          >
            Remove from this {entity.type}
          </Button>
        }
      />
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {explanation &&
        groupActions(actions).map((group) => (
          <RuleGroup key={group.name} userId={userId} group={group} explanation={explanation} catalog={catalog} />
        ))}
    </div>
  );
};

// The chosen person's rule here, grouped by domain; nobody chosen is one muted sentence.
export const RulesPane = () => {
  const entity = usePanelEntity();
  const target = usePermissionPanelStore((s) => s.target);
  return (
    <div className="min-w-0">
      {!target && <EmptyRow flush>Pick a person, or add one, to set rules on {entity.name}.</EmptyRow>}
      {target && <RulesFor key={target} userId={target} />}
    </div>
  );
};
