import { ChevronRightIcon } from "lucide-react";
import { useSearchParams } from "react-router";

import { Button } from "@/components/ui/button";
import { useListComputers } from "@/hooks/PairingHooks";
import { cn } from "@/lib/utils";
import { COMPUTERS_PARAM } from "@/models/Agent";

// Opens the Computers dialog, which reads `?computers=`; the dot and "2 of 4 online" say how many of the finished computers are online.
export const ComputersButton = () => {
  const [, setParams] = useSearchParams();
  const { data } = useListComputers();
  const computers = data?.filter((c) => !c.unfinished);
  const online = computers?.filter((c) => c.runner?.connected).length ?? 0;

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
      <span aria-hidden className={cn("size-1.5 rounded-full", online > 0 ? "bg-success" : "bg-muted-foreground/50")} />
      Computers
      {computers && (
        <span className="text-xs text-muted-foreground tabular-nums">
          {online} of {computers.length} online
        </span>
      )}
      <ChevronRightIcon aria-hidden className="size-3.5 text-muted-foreground" />
    </Button>
  );
};
