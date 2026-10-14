import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PanelPaneHeader } from "@/components/access/PanelPaneHeader";
import { PermissionGroupSection } from "@/components/access/PermissionGroupSection";
import { RequiresLine } from "@/components/access/RequiresLine";
import { SourceLine } from "@/components/access/SourceLine";
import { useFetchExplanation, useFetchPermissionCatalog } from "@/hooks/PermissionHooks";
import { usePanelEntity } from "@/hooks/usePanelEntity";
import { usePanelTargets } from "@/hooks/usePanelTargets";
import { usePermissionPanelStore } from "@/stores/permissionPanelStore";
import type { PermissionAnswer, PermissionInfo } from "@/models/Permission";
import { entityWord, groupActions, permissionLabel, targetKey, targetScope, type PanelEntity, type PanelGroup } from "@/models/PermissionPanel";

interface CheckGroupProps {
  type: PanelEntity["type"];
  group: PanelGroup;
  answers: PermissionAnswer[];
  catalog: PermissionInfo[] | undefined;
}

const CheckGroup = ({ type, group, answers, catalog }: CheckGroupProps) => (
  <PermissionGroupSection name={group.name}>
    {answers
      .filter((a) => group.actions.includes(a.action))
      .map((answer) => (
        <li key={answer.action} className="px-3 py-2.5">
          <p className="text-sm">{permissionLabel(answer.action, catalog, type)}</p>
          <SourceLine verdict={answer} className="mt-1" />
          <RequiresLine answer={answer} />
        </li>
      ))}
  </PermissionGroupSection>
);

const CheckFor = ({ userId }: { userId: string }) => {
  const entity = usePanelEntity();
  const { targetOf } = usePanelTargets(targetScope(entity));
  const { data: catalog } = useFetchPermissionCatalog();
  const { data: explanation, error, isPending } = useFetchExplanation(entity.type, entity.id, userId);
  const person = targetOf(targetKey("person", userId));
  const answers = explanation?.effective ?? [];
  // Outside the entity's project every answer is the same not found, said once.
  const outside = answers.length > 0 && answers.every((a) => a.source.kind === "not_in_project") ? answers[0] : undefined;

  return (
    <div className="min-w-0 flex-1 space-y-5">
      <PanelPaneHeader
        target={person}
        title={`What ${person.name} can do`}
        line={`On ${entityWord(entity.type)} ${entity.name}, read from every rule that reaches it`}
      />
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {outside && <SourceLine verdict={outside} className="rounded-md border border-input px-3 py-3 text-sm" />}
      {explanation &&
        !outside &&
        groupActions(answers.map((a) => a.action), catalog).map((group) => (
          <CheckGroup key={group.name} type={entity.type} group={group} answers={answers} catalog={catalog} />
        ))}
    </div>
  );
};

// One person's answer for every permission here, each with the sentence that decided it.
export const CheckAccessPane = () => {
  const entity = usePanelEntity();
  const checked = usePermissionPanelStore((s) => s.checked);
  return (
    <div className="min-w-0">
      {!checked && <EmptyRow flush>Pick a person to see what they can do on {entity.name}.</EmptyRow>}
      {checked && <CheckFor key={checked} userId={checked} />}
    </div>
  );
};
