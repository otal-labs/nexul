import { Link } from "react-router";

import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PairingProjectsFeed } from "@/components/settings/PairingProjectsFeed";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { useListComputers } from "@/hooks/PairingHooks";
import { useFetchProjectLinks } from "@/hooks/PairingProjectHooks";
import { useTabPath } from "@/hooks/useTabPath";
import { useFetchWorkspaces } from "@/hooks/WorkspaceHooks";

// Waits for the links too, so no row reads "Uses your defaults" before its own link has loaded.
export const PairingProjectsSection = () => {
  const { tabPath } = useTabPath();
  const { data: computers, isPending: computersPending, error: computersError } = useListComputers();
  const { data: workspaces, isPending: workspacesPending, error: workspacesError } = useFetchWorkspaces();
  const { data: links, isPending: linksPending, error: linksError } = useFetchProjectLinks();
  const isPending = computersPending || workspacesPending || linksPending;
  const error = computersError || workspacesError || linksError;

  return (
    <SettingsCard
      id="pairing-projects"
      title="Projects"
      description="The computer, T3 project, and model your @Agent turns use in each project. Nobody else's turns use them, and a project you leave unlinked uses your defaults."
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {computers && computers.length === 0 && (
        <EmptyRow>
          Pair a computer on the{" "}
          <Link to={tabPath()} className="text-foreground underline underline-offset-4">
            Computers
          </Link>{" "}
          tab first, then link it to your projects here.
        </EmptyRow>
      )}
      {computers && computers.length > 0 && workspaces && links && <PairingProjectsFeed workspaces={workspaces} />}
    </SettingsCard>
  );
};
