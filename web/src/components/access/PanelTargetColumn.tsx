import { unknownPerson } from "@nexul/client-core/person";

import { Microheader } from "@/components/Microheader";
import { PanelAddPerson } from "@/components/access/PanelAddPerson";
import { PanelTargetRow } from "@/components/access/PanelTargetRow";
import { useFetchWorkspacePeople } from "@/hooks/PeopleHooks";
import { usePanelEntity } from "@/hooks/usePanelEntity";
import { usePermissionRules } from "@/hooks/usePermissionRules";
import { usePermissionPanelStore } from "@/stores/permissionPanelStore";
import { rulesWord } from "@/models/PermissionPanel";

const listClass = "flex gap-1 @3xl/panel:flex-col @3xl/panel:gap-0";
const headClass = "hidden px-3 pt-1 pb-2 @3xl/panel:block";

// Rules lists the people with a rule here, the add control at the foot; Check access lists every member.
export const PanelTargetColumn = () => {
  const entity = usePanelEntity();
  const mode = usePermissionPanelStore((s) => s.mode);
  const target = usePermissionPanelStore((s) => s.target);
  const checked = usePermissionPanelStore((s) => s.checked);
  const pick = usePermissionPanelStore((s) => s.pick);
  const check = usePermissionPanelStore((s) => s.check);
  const { data: people } = useFetchWorkspacePeople(entity.workspaceId);
  const { listed, ruleCount } = usePermissionRules();
  const personOf = (userId: string) => people?.find((p) => p.user_id === userId) ?? unknownPerson(userId);

  return (
    <nav
      aria-label={mode === "check" ? "People to check" : "People with rules"}
      className="flex min-w-0 shrink-0 items-center gap-1 overflow-x-auto pb-1 @3xl/panel:w-60 @3xl/panel:flex-col @3xl/panel:items-stretch @3xl/panel:gap-0 @3xl/panel:overflow-x-hidden @3xl/panel:overflow-y-auto @3xl/panel:border-r @3xl/panel:border-border @3xl/panel:py-2"
    >
      {mode === "check" && <Microheader className={headClass}>Check a person</Microheader>}
      {mode === "check" && (
        <ul className={listClass}>
          {(people ?? []).map((person) => (
            <PanelTargetRow key={person.user_id} person={person} selected={checked === person.user_id} onSelect={() => check(person.user_id)} subline={person.login} />
          ))}
        </ul>
      )}
      {mode === "rules" && listed.length > 0 && <Microheader className={headClass}>People</Microheader>}
      {mode === "rules" && (
        <ul className={listClass}>
          {listed.map((userId) => (
            <PanelTargetRow key={userId} person={personOf(userId)} selected={target === userId} onSelect={() => pick(userId)} subline={rulesWord(ruleCount(userId))} />
          ))}
        </ul>
      )}
      {mode === "rules" && (
        <div className="shrink-0 @3xl/panel:mt-auto @3xl/panel:px-1.5 @3xl/panel:pt-3">
          <PanelAddPerson listed={listed} />
        </div>
      )}
    </nav>
  );
};
