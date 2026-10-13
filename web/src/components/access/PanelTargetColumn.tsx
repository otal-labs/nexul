import { Microheader } from "@/components/Microheader";
import { PanelAddTarget } from "@/components/access/PanelAddTarget";
import { PanelTargetRow } from "@/components/access/PanelTargetRow";
import { usePanelEntity } from "@/hooks/usePanelEntity";
import { usePanelTargets } from "@/hooks/usePanelTargets";
import { usePermissionRules } from "@/hooks/usePermissionRules";
import { usePermissionPanelStore } from "@/stores/permissionPanelStore";
import { rulesWord, targetScope } from "@/models/PermissionPanel";

const listClass = "flex gap-1 @3xl/panel:flex-col @3xl/panel:gap-0";
const headClass = "hidden px-3 pt-1 pb-2 @3xl/panel:block";

// Rules lists the roles, then the people, with a rule here, the add control at the foot; Check access lists every member.
export const PanelTargetColumn = () => {
  const entity = usePanelEntity();
  const mode = usePermissionPanelStore((s) => s.mode);
  const target = usePermissionPanelStore((s) => s.target);
  const checked = usePermissionPanelStore((s) => s.checked);
  const pick = usePermissionPanelStore((s) => s.pick);
  const check = usePermissionPanelStore((s) => s.check);
  const { people, targetOf } = usePanelTargets(targetScope(entity));
  const { listed, roles, people: listedPeople, ruleCount } = usePermissionRules();

  return (
    <nav
      aria-label={mode === "check" ? "People to check" : "Roles and people with rules"}
      className="flex min-w-0 shrink-0 items-center gap-1 overflow-x-auto pb-1 @3xl/panel:w-60 @3xl/panel:flex-col @3xl/panel:items-stretch @3xl/panel:gap-0 @3xl/panel:overflow-x-hidden @3xl/panel:overflow-y-auto @3xl/panel:border-r @3xl/panel:border-border @3xl/panel:py-2"
    >
      {mode === "check" && <Microheader className={headClass}>Check a person</Microheader>}
      {mode === "check" && (
        <ul className={listClass}>
          {people.map((person) => (
            <PanelTargetRow key={person.key} target={person} selected={checked === person.id} onSelect={() => check(person.id)} subline={person.login} />
          ))}
        </ul>
      )}
      {mode === "rules" && roles.length > 0 && <Microheader className={headClass}>Roles</Microheader>}
      {mode === "rules" && roles.length > 0 && (
        <ul className={listClass}>
          {roles.map((key) => (
            <PanelTargetRow key={key} target={targetOf(key)} selected={target === key} onSelect={() => pick(key)} subline={rulesWord(ruleCount(key))} />
          ))}
        </ul>
      )}
      {mode === "rules" && roles.length > 0 && listedPeople.length > 0 && <span className="h-5 w-px shrink-0 bg-border @3xl/panel:hidden" aria-hidden />}
      {mode === "rules" && listedPeople.length > 0 && <Microheader className={`${headClass} @3xl/panel:pt-3`}>People</Microheader>}
      {mode === "rules" && listedPeople.length > 0 && (
        <ul className={listClass}>
          {listedPeople.map((key) => (
            <PanelTargetRow key={key} target={targetOf(key)} selected={target === key} onSelect={() => pick(key)} subline={rulesWord(ruleCount(key))} />
          ))}
        </ul>
      )}
      {mode === "rules" && (
        <div className="shrink-0 @3xl/panel:mt-auto @3xl/panel:px-1.5 @3xl/panel:pt-3">
          <PanelAddTarget listed={listed} />
        </div>
      )}
    </nav>
  );
};
