import { useState } from "react";
import { PlusIcon } from "lucide-react";

import { AddProjectSection } from "@/components/settings/AddProjectSection";
import { ProjectCheckoutCell } from "@/components/settings/ProjectCheckoutCell";
import { Button } from "@/components/ui/button";
import { useListComputers } from "@/hooks/PairingHooks";
import { useFetchProjectLinks, useSetProjectLink } from "@/hooks/PairingProjectHooks";
import { useFetchProjectRepos, useFetchProjects } from "@/hooks/ProjectHooks";
import { projectLinkFormValues, type PassOver } from "@/models/Pairing";
import { projectRemotes, repoCheckouts, withCheckout } from "@/models/ProjectCheckout";

interface WaitingCheckoutFixProps {
  projectId: string;
  pass: PassOver;
}

// The fix for a run held on one computer's checkout, on the run itself: pick between its checkouts, or add the project
// there. The pick is saved on the project's link, which wakes the waiting run (spec decision 6).
export const WaitingCheckoutFix = ({ projectId, pass }: WaitingCheckoutFixProps) => {
  const [adding, setAdding] = useState(false);
  const { data: computers } = useListComputers();
  const { data: repos } = useFetchProjectRepos(projectId);
  const { data: links } = useFetchProjectLinks();
  const { data: projects } = useFetchProjects();
  const projectName = projects?.find((p) => p.id === projectId)?.name ?? "The project";
  const save = useSetProjectLink(projectId, projectName);
  const computer = computers?.find((c) => c.id === pass.computer_id);
  if (!computer || !repos || !links) return null;

  const link = links.find((l) => l.project_id === projectId) ?? { project_id: projectId, computers: [], checkouts: [] };
  const checkouts = repoCheckouts(computer, link.checkouts, projectRemotes(repos));
  const pick = (repository: string, harnessProjectId: string) => {
    const values = projectLinkFormValues(link);
    save.mutate({ ...values, checkouts: withCheckout(values.checkouts, computer.id, repository.toLowerCase(), harnessProjectId) });
    setAdding(false);
  };
  const ambiguous = checkouts.filter((c) => c.checkout.state === "ambiguous");
  const missing = checkouts.find((c) => c.checkout.state === "missing");

  return (
    <div className="mt-1.5 space-y-2">
      {pass.why === "ambiguous_checkout" &&
        ambiguous.map((c) => (
          <ProjectCheckoutCell
            key={c.repository}
            computerName={computer.name}
            repository={checkouts.length > 1 ? c.repository : undefined}
            checkout={c.checkout}
            onPick={(id) => pick(c.repository, id)}
            className="justify-start"
          />
        ))}
      {pass.why !== "ambiguous_checkout" && missing && !adding && (
        <Button type="button" variant="outline" size="sm" onClick={() => setAdding(true)}>
          <PlusIcon className="size-4" aria-hidden />
          Add it to {computer.name}
        </Button>
      )}
      {pass.why !== "ambiguous_checkout" && missing && adding && (
        <AddProjectSection projectId={projectId} computerId={computer.id} projects={computer.facts?.projects ?? []} onAdded={(id) => pick(missing.repository, id)} />
      )}
    </div>
  );
};
