import { ChevronRightIcon } from "lucide-react";
import { useSearchParams } from "react-router";

import { Button } from "@/components/ui/button";
import { useFetchPresence, useListComputers } from "@/hooks/PairingHooks";
import { cn } from "@/lib/utils";
import { COMPUTERS_PARAM } from "@/models/Agent";
import { computerReadiness } from "@/models/ComputerChecks";

// Opens the Computers dialog, which reads `?computers=`; the dot and "2 of 4 ready" say how many of the finished computers can take a run.
export const ComputersButton = () => {
  const [, setParams] = useSearchParams();
  const { data } = useListComputers();
  const presence = useFetchPresence();
  const computers = data?.filter((c) => !c.unfinished);
  const ready = computers?.filter((c) => computerReadiness(c, presence.data?.[c.id]).state === "ready").length ?? 0;

  return (
    <Button
      type="button"
      variant="outline"
      onClick={() =>
        setParams((current) => {
          const next = new URLSearchParams(current);
          next.set(COMPUTERS_PARAM, "");
          return next;
        })
      }
    >
      <span aria-hidden className={cn("size-1.5 rounded-full", ready > 0 ? "bg-success" : "bg-muted-foreground/50")} />
      Computers
      {computers && (
        <span className="text-xs text-muted-foreground tabular-nums">
          {ready} of {computers.length} ready
        </span>
      )}
      <ChevronRightIcon aria-hidden className="size-3.5 text-muted-foreground" />
    </Button>
  );
};
