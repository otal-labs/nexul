import { ChevronLeftIcon } from "lucide-react";
import { useState, type MouseEvent } from "react";

import { ComputerPane } from "@/components/computers/ComputerPane";
import { ComputersList } from "@/components/computers/ComputersList";
import { DefaultsRunPane } from "@/components/computers/DefaultsRunPane";
import { ProjectRunPane } from "@/components/computers/ProjectRunPane";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { Button } from "@/components/ui/button";
import { useListOrderComputers } from "@/hooks/PairingHooks";
import { cn } from "@/lib/utils";
import { DEFAULTS_ID, PROJECT_TARGET_PREFIX } from "@/models/Agent";

interface ComputersBodyProps {
  // The `computers` param: "" the first computer, a computer's id, "defaults", or "project:<id>".
  target: string;
}

// The list beside the open record; each column scrolls on its own. Below 640px one of them shows at a time: the list when
// the dialog opens on no record, then the record with a back control.
export const ComputersBody = ({ target }: ComputersBodyProps) => {
  const { data: computers } = useListOrderComputers();
  // The target the list was last asked for; picking a row sends the person to the record, whatever it is.
  const [listFor, setListFor] = useState<string | null>(target === "" ? "" : null);
  const listShown = listFor === target;
  const current = target === "" ? (computers?.[0]?.id ?? "") : target;
  const isProject = current.startsWith(PROJECT_TARGET_PREFIX);
  const isComputer = current !== "" && current !== DEFAULTS_ID && !isProject;

  const onListClick = (e: MouseEvent) => {
    if ((e.target as HTMLElement).closest("a")) setListFor(null);
  };

  return (
    <div className="grid min-h-0 flex-1 grid-cols-1 grid-rows-[minmax(0,1fr)] sm:grid-cols-[16rem_minmax(0,1fr)] lg:grid-cols-[18rem_minmax(0,1fr)]">
      {/* A scroller of its own with the app's scrollbar and room under its last row; nothing fades over it. */}
      <nav
        aria-label="Computers and run settings"
        tabIndex={-1}
        onClick={onListClick}
        className={cn("min-h-0 overflow-y-auto border-border pb-6 outline-none sm:border-r", !listShown && "max-sm:hidden")}
      >
        <ComputersList current={current} />
      </nav>
      <section aria-label="Details" className={cn("min-h-0 overflow-y-auto px-4 pt-4 pb-6 sm:px-6 sm:pt-6", listShown && "max-sm:hidden")}>
        <Button type="button" variant="ghost" size="sm" className="-ml-2 mb-3 sm:hidden" onClick={() => setListFor(target)}>
          <ChevronLeftIcon aria-hidden />
          Computers
        </Button>
        <div className="space-y-6">
          {isComputer && <ComputerPane key={current} computerId={current} />}
          {current === DEFAULTS_ID && <DefaultsRunPane />}
          {isProject && <ProjectRunPane key={current} projectId={current.slice(PROJECT_TARGET_PREFIX.length)} />}
          {computers && computers.length === 0 && current === "" && <NoDataDisplay message="No computers yet. Add one to run plays on it." />}
        </div>
      </section>
    </div>
  );
};
