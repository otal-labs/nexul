import { LockIcon } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { WizardFooter } from "@/components/wizard/WizardFooter";
import { WizardRoleRow } from "@/components/wizard/WizardRoleRow";
import { WizardSkipLink } from "@/components/wizard/WizardSkipLink";
import { useFetchGrants, useSaveRules } from "@/hooks/PermissionHooks";
import { useActionState } from "@/hooks/useActionState";
import { useEntityActions } from "@/hooks/useEntityActions";
import { usePanelTargets } from "@/hooks/usePanelTargets";
import { useCanOpenSection } from "@/hooks/AccessHooks";
import { useFetchTeam } from "@/hooks/TeamHooks";
import { RuleState } from "@/models/Permission";
import { targetKey } from "@/models/PermissionPanel";
import { holdItWord, IN_THIS_PROJECT, projectPeopleRules } from "@/models/ProjectPeople";
import type { Project } from "@/models/Project";
import { roleHolders } from "@/models/Team";

interface WizardPeopleStepProps {
  project: Project;
  onDone: () => void;
  onBack: () => void;
  onSkip: () => void;
}

// "Who's in it": the roles the project lets in, pre-ticked with the creator's own by its creation; the creator is always in.
export const WizardPeopleStep = ({ project, onDone, onBack, onSkip }: WizardPeopleStepProps) => {
  const { data: grants, error, isPending } = useFetchGrants("project", project.id);
  const { roles, ready } = usePanelTargets({ workspaceId: project.workspace_id });
  // Holder counts come from Team, so only a viewer who may read Team asks for it.
  const { data: team } = useFetchTeam(useCanOpenSection("team"));
  const save = useSaveRules();
  const answer = useEntityActions("project", project.id);
  const manage = useActionState({ permission: "projects:access", projectId: project.id, answer: answer("projects:access"), verb: "choose who is in a project" });
  // Ticks the person changed here, over what the project's rules already list.
  const [ticked, setTicked] = useState<Record<string, boolean>>({});
  const listed = grants ? projectPeopleRules(grants).roles.map((g) => g.role_id ?? "") : [];
  const isTicked = (id: string) => ticked[id] ?? listed.includes(id);
  const holders = (id: string) => (team ? holdItWord(roleHolders(team, project.workspace_id, id).length) : "Its holders can open the project");

  const onContinue = async () => {
    const moved = roles.filter((r) => isTicked(r.id) !== listed.includes(r.id));
    const changes = moved.map((r) => ({ target: targetKey("role", r.id), state: isTicked(r.id) ? RuleState.Allow : RuleState.Fallback, actions: [IN_THIS_PROJECT] }));
    try {
      await save.mutateAsync({ resourceType: "project", resourceId: project.id, changes, done: `Saved who's in ${project.name}` });
      onDone();
    } catch {
      // The hook shows the error; the step stays open for retry.
    }
  };

  return (
    <div className="space-y-4">
      {(isPending || !ready) && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {grants && ready && roles.length === 0 && <p className="text-sm text-muted-foreground">This workspace has no roles but the Owner's yet.</p>}
      {manage.state === "locked" && (
        <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
          <LockIcon className="size-3.5 shrink-0" aria-hidden />
          {manage.reason}
        </p>
      )}
      {grants && ready && roles.length > 0 && (
        <ul className="divide-y divide-border border-y border-border" aria-label="Roles in this project">
          {roles.map((role) => (
            <WizardRoleRow key={role.key} role={role} checked={isTicked(role.id)} line={holders(role.id)} locked={manage.state !== "live"} onToggle={(on) => setTicked((t) => ({ ...t, [role.id]: on }))} />
          ))}
        </ul>
      )}
      <WizardFooter onBack={onBack} skip={<WizardSkipLink onClick={onSkip} />}>
        <Button type="button" disabled={!grants} loading={save.isPending} onClick={() => void onContinue()}>
          Continue
        </Button>
      </WizardFooter>
    </div>
  );
};
