import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { SignInAccounts } from "@/components/SignInAccounts";
import { ComputerShareWarning } from "@/components/access/ComputerShareWarning";
import { PanelPaneHeader } from "@/components/access/PanelPaneHeader";
import { PermissionGroupSection } from "@/components/access/PermissionGroupSection";
import { PermissionRuleRow } from "@/components/access/PermissionRuleRow";
import { useFetchExplanation, useFetchPermissionCatalog } from "@/hooks/PermissionHooks";
import { usePanelEntity } from "@/hooks/usePanelEntity";
import { usePanelTargets } from "@/hooks/usePanelTargets";
import { usePermissionRules } from "@/hooks/usePermissionRules";
import { usePermissionPanelStore } from "@/stores/permissionPanelStore";
import type { PermissionExplanation, PermissionInfo } from "@/models/Permission";
import { groupActions, parseTarget, permissionLabel, targetScope, type PanelEntity, type PanelGroup } from "@/models/PermissionPanel";

// The role behind a person's answers, read from the sources that name one.
const heldRole = (explanation: PermissionExplanation | undefined) =>
  explanation?.effective.map((a) => a.fallback.source).find((s) => s.target_type === "role" && (s.kind === "role" || s.kind === "owner"))?.target_name;

// The explain request's target for a rule's target key: a role by "role:<id>", a person by their id.
const explainTargetOf = (key: string) => {
  const { kind, id } = parseTarget(key);
  return kind === "role" ? key : id;
};

interface RuleGroupProps {
  targetKey: string;
  type: PanelEntity["type"];
  group: PanelGroup;
  explanation: PermissionExplanation;
  catalog: PermissionInfo[] | undefined;
}

const RuleGroup = ({ targetKey, type, group, explanation, catalog }: RuleGroupProps) => (
  <PermissionGroupSection name={group.name}>
    {group.actions.map((action) => (
      <PermissionRuleRow
        key={action}
        targetKey={targetKey}
        action={action}
        label={permissionLabel(action, catalog, type)}
        answer={explanation.effective.find((a) => a.action === action)}
      />
    ))}
  </PermissionGroupSection>
);

const RulesFor = ({ targetKey }: { targetKey: string }) => {
  const entity = usePanelEntity();
  const { targetOf } = usePanelTargets(targetScope(entity));
  const { data: catalog } = useFetchPermissionCatalog();
  const { data: explanation, error, isPending } = useFetchExplanation(entity.type, entity.id, explainTargetOf(targetKey));
  const { actions } = usePermissionRules();
  const target = targetOf(targetKey);
  const role = heldRole(explanation);
  const roleLine = role ? `holds ${role}` : "holds no role";
  // A computer's rules never read roles, so its person is named by their sign-in accounts instead.
  const personLine = entity.type === "computer" ? <SignInAccounts providers={target.providers} usernames={target.usernames} /> : roleLine;
  const line = target.kind === "role" ? "Role" : <>Person · {personLine}</>;

  return (
    <div className="min-w-0 flex-1 space-y-5">
      <PanelPaneHeader target={target} title={target.name} line={line} />
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {explanation &&
        groupActions(actions, catalog).map((group) => (
          <RuleGroup key={group.name} targetKey={targetKey} type={entity.type} group={group} explanation={explanation} catalog={catalog} />
        ))}
    </div>
  );
};

// The chosen target's rule here, grouped by domain; nothing chosen is one muted sentence. A computer leads with its warning.
export const RulesPane = () => {
  const entity = usePanelEntity();
  const target = usePermissionPanelStore((s) => s.target);
  const computer = entity.type === "computer";
  return (
    <div className="min-w-0 space-y-5">
      {computer && <ComputerShareWarning />}
      {!target && !computer && <EmptyRow flush>Pick a role or person, or add one, to set rules on {entity.name}.</EmptyRow>}
      {!target && computer && <EmptyRow flush>Add a person to share {entity.name} with them.</EmptyRow>}
      {target && <RulesFor key={target} targetKey={target} />}
    </div>
  );
};
