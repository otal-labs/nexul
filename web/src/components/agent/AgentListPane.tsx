import { useState } from "react";
import { MonitorUpIcon } from "lucide-react";

import { AgentComputersGroup } from "@/components/agent/AgentComputersGroup";
import { AgentProjectsGroup } from "@/components/agent/AgentProjectsGroup";
import { AgentSharedGroup } from "@/components/agent/AgentSharedGroup";
import { ListPaneHeader } from "@/components/listpane/ListPaneHeader";
import { AddComputerDialog } from "@/components/pairing/AddComputerDialog";
import { Button } from "@/components/ui/button";

// Your computers, the ones shared with you, and project run settings; the Computers dialog replaces it.
export const AgentListPane = () => {
  const [search, setSearch] = useState("");
  const query = search.trim().toLowerCase();

  return (
    <div className="flex h-full min-h-0 flex-col">
      <ListPaneHeader
        title="Agent"
        count={0}
        hideZeroCount
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
        <AgentComputersGroup query={query} />
        <AgentSharedGroup query={query} />
        <AgentProjectsGroup query={query} />
      </div>
    </div>
  );
};
