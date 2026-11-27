import { useState } from "react";
import { MonitorUpIcon } from "lucide-react";

import { AgentPlayGroups } from "@/components/agent/AgentPlayGroups";
import { ListPaneHeader } from "@/components/listpane/ListPaneHeader";
import { AddComputerDialog } from "@/components/pairing/AddComputerDialog";
import { Button } from "@/components/ui/button";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { usePlayDialogs } from "@/hooks/usePlayDialogs";

// What the Agent runs: the play groups. Computers and run settings open in the Computers dialog.
export const AgentListPane = () => {
  const [search, setSearch] = useState("");
  const can = useAreaAccess();
  const { create } = usePlayDialogs();
  const query = search.trim().toLowerCase();
  const plays = !!can?.("plays");

  return (
    <div className="flex h-full min-h-0 flex-col">
      <ListPaneHeader
        title="Agent"
        count={0}
        hideZeroCount
        {...(plays && { newLabel: "New play", newPermission: "plays:write" as const, newVerb: "create plays", onNew: () => void create() })}
        search={search}
        onSearch={setSearch}
        controls={
          <AddComputerDialog
            trigger={
              <Button type="button" variant="ghost" size="icon" className="size-8" aria-label="Add a computer" title="Add a computer">
                <MonitorUpIcon className="size-4" aria-hidden />
              </Button>
            }
          />
        }
      />
      <div className="scroll-edge min-h-0 flex-1 overflow-y-auto pb-6 [scrollbar-gutter:stable_both-edges]">
        {plays && <AgentPlayGroups query={query} />}
      </div>
    </div>
  );
};
