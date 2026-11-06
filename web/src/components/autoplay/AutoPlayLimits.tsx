import { ComputerRunsRow } from "@/components/autoplay/ComputerRunsRow";
import { DailyCapRow } from "@/components/autoplay/DailyCapRow";
import { Microheader } from "@/components/Microheader";
import { useListComputers } from "@/hooks/PairingHooks";
import { useFetchAutoPlayLimits } from "@/hooks/PlayHooks";
import { cn } from "@/lib/utils";

interface AutoPlayLimitsProps {
  workspaceId: string;
  /** Only ticket plays carry the daily cap. */
  ticket: boolean;
  canWrite: boolean;
}

// What bounds the auto plays beyond this play: the workspace's daily cap and the viewer's own computers' runs at once.
// Only a computer's owner sets its runs at once, so only the viewer's computers are listed, whatever they may do to the play.
export const AutoPlayLimits = ({ workspaceId, ticket, canWrite }: AutoPlayLimitsProps) => {
  const { data: limits } = useFetchAutoPlayLimits(ticket ? workspaceId : "");
  const { data: computers } = useListComputers();
  const cap = limits?.daily_cap_per_ticket;
  const mine = computers ?? [];

  if (cap === undefined && mine.length === 0) return null;
  return (
    <div className="space-y-3 pt-3">
      <Microheader className="flex h-7 items-center">Limits</Microheader>
      <div className="rounded-lg border border-transparent bg-surface-2 px-4 py-1">
        {cap !== undefined && <DailyCapRow workspaceId={workspaceId} cap={cap} canWrite={canWrite} />}
        {mine.length > 0 && (
          <div className={cn("pt-3", cap !== undefined && "mt-1 border-t border-border")}>
            <Microheader className="pb-1">Your computers · runs at once</Microheader>
            {mine.map((computer) => (
              <ComputerRunsRow key={computer.id} computer={computer} />
            ))}
          </div>
        )}
      </div>
    </div>
  );
};
