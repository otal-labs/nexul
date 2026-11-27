import { Navigate, useLocation, useParams } from "react-router";

import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useFetchWorkspacePlays } from "@/hooks/PlayHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { agentPlayPath, agentTabPath, computersDialogPath, DEFAULTS_ID } from "@/models/Agent";
import { useWorkspaceStore } from "@/stores/workspaceStore";

// Links from before the tabs: /agent/plays/<id>, /agent/computers/<id> and /agent/projects/<id> land where their content lives now.
export const AgentLegacyRedirect = () => {
  const { kind, id = "" } = useParams();
  const { search } = useLocation();
  const wsPath = useWorkspacePath();
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: plays, error, isPending } = useFetchWorkspacePlays(kind === "plays" ? workspaceId : "");
  const play = plays?.find((p) => p.id === id);
  const setup = new URLSearchParams(search).get("setup");

  return (
    <>
      {kind === "plays" && isPending && !error && <LoadingDisplay label="Opening the play…" />}
      {kind === "plays" && play && <Navigate replace to={wsPath(agentPlayPath(play.type, play.id))} />}
      {kind === "plays" && !play && (!isPending || !!error) && <Navigate replace to={wsPath(agentTabPath("ticket"))} />}
      {kind === "computers" && (
        <Navigate replace to={wsPath(`${computersDialogPath(id)}${setup ? `&setup=${encodeURIComponent(setup)}` : ""}`)} />
      )}
      {kind === "projects" && <Navigate replace to={wsPath(computersDialogPath(id === DEFAULTS_ID ? DEFAULTS_ID : `project:${id}`))} />}
      {kind !== "plays" && kind !== "computers" && kind !== "projects" && <Navigate replace to={wsPath(agentTabPath("ticket"))} />}
    </>
  );
};
