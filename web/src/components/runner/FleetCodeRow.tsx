import { MoreHorizontalIcon, XIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { RunnerName } from "@/components/runner/RunnerName";
import { fleetTracks } from "@/components/runner/fleetTracks";
import { useRevokeEnrollment } from "@/hooks/RunnerHooks";
import type { OpenEnrollment } from "@/models/Runner";
import { cn } from "@/lib/utils";
import { formatClockTime } from "@/utils/TimeUtility";

interface FleetCodeRowProps {
  code: OpenEnrollment;
}

// A runner not yet added: its code works once, until the time shown, which is the last thing to truncate.
export const FleetCodeRow = ({ code }: FleetCodeRowProps) => {
  const revoke = useRevokeEnrollment();
  const until = formatClockTime(code.expires_at);
  return (
    <li className={cn(fleetTracks, "items-baseline gap-x-4 px-4 py-2.5 leading-5 transition-colors duration-150 ease-standard hover:bg-accent/40")}>
      <span className="flex min-w-0 items-baseline gap-3">
        <span className="flex self-center" title="Not added yet">
          <span aria-hidden className="size-2 rounded-full border border-dashed border-muted-foreground" />
        </span>
        <RunnerName name={code.name} muted />
      </span>
      <span />
      <span className="flex min-w-0 items-baseline text-[13px] text-muted-foreground" title={`Works once, until ${until}`}>
        <span className="truncate">unused code · until&nbsp;</span>
        <span className="shrink-0 font-mono text-xs tabular-nums">{until}</span>
      </span>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="icon" className="-my-1 size-7 self-center justify-self-end" aria-label={`More actions for the code for ${code.name}`} loading={revoke.isPending}>
            <MoreHorizontalIcon className="size-4" aria-hidden />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="min-w-44">
          <DropdownMenuItem onSelect={() => revoke.mutate(code.id)}>
            <XIcon aria-hidden />
            Revoke code
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </li>
  );
};
