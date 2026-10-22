import { arrayMove } from "@dnd-kit/sortable";
import { useFormContext, useWatch } from "react-hook-form";

import { ProjectComputerRow } from "@/components/settings/ProjectComputerRow";
import { SettingsRow } from "@/components/settings/SettingsRow";
import { FormSelect } from "@/components/ticket/FormSelect";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { useListComputers } from "@/hooks/PairingHooks";
import { useFetchProjectRepos } from "@/hooks/ProjectHooks";
import { PLACEMENT_OPTIONS, type Computer, type ProjectLinkFormData } from "@/models/Pairing";
import { projectCheckout, projectRemotes, withCheckout } from "@/models/ProjectCheckout";

interface ProjectComputersFieldProps {
  projectId: string;
}

// The computers a project's runs try, ranked: the person's order, or the project's own list in its own order, each with
// the checkout a run would use there (ADR 0165).
export const ProjectComputersField = ({ projectId }: ProjectComputersFieldProps) => {
  const { control, setValue } = useFormContext<ProjectLinkFormData>();
  const ids = useWatch({ control, name: "computers" });
  const checkouts = useWatch({ control, name: "checkouts" });
  const { data: own = [] } = useListComputers();
  const { data: repos = [] } = useFetchProjectRepos(projectId);
  const remotes = projectRemotes(repos);
  const ownOrder = ids.length > 0;
  const listed = ownOrder ? ids.flatMap((id) => own.filter((c) => c.id === id)) : own;
  const rest = own.filter((c) => !ids.includes(c.id));
  const dirty = { shouldDirty: true };

  const setIds = (next: string[]) => setValue("computers", next, dirty);
  const actionsFor = (computer: Computer, index: number) =>
    ownOrder
      ? [
          { label: "Move up", disabled: index === 0, onSelect: () => setIds(arrayMove(ids, index, index - 1)) },
          { label: "Move down", disabled: index === ids.length - 1, onSelect: () => setIds(arrayMove(ids, index, index + 1)) },
          { label: "Remove from this project", disabled: ids.length === 1, onSelect: () => setIds(ids.filter((id) => id !== computer.id)) },
        ]
      : [];

  return (
    <div className="space-y-3">
      <SettingsRow
        label="Its own computers"
        description={ownOrder ? "Runs here try only these, in this order." : "Runs here follow your order of computers."}
        htmlFor={`own-order-${projectId}`}
        className="py-0"
      >
        <Switch id={`own-order-${projectId}`} checked={ownOrder} onCheckedChange={(on) => setIds(on ? own.map((c) => c.id) : [])} />
      </SettingsRow>
      <ul aria-label="Computers this project's runs try" className="ml-10 divide-y divide-border rounded-md border">
        {listed.map((computer, index) => (
          <ProjectComputerRow
            key={computer.id}
            projectId={projectId}
            computer={computer}
            rank={index + 1}
            checkout={projectCheckout(computer, checkouts, remotes)}
            actions={actionsFor(computer, index)}
            onPick={(harnessProjectId) => setValue("checkouts", withCheckout(checkouts, computer.id, harnessProjectId), dirty)}
          />
        ))}
      </ul>
      {ownOrder && rest.length > 0 && (
        <Select value="" onValueChange={(id) => setIds([...ids, id])}>
          <SelectTrigger aria-label="Add a computer to this project" className="ml-10 h-8 w-52 text-xs">
            <SelectValue placeholder="Add a computer" />
          </SelectTrigger>
          <SelectContent>
            {rest.map((c) => (
              <SelectItem key={c.id} value={c.id} className="text-xs">
                {c.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      )}
      {ownOrder && (
        <FormSelect control={control} name="placement" label="When several can take a run" placeholder="Same as your defaults" options={PLACEMENT_OPTIONS} />
      )}
    </div>
  );
};
