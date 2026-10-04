import { HandoffPill } from "@/components/handoff/HandoffPill";
import type { Handoff } from "@/models/Handoff";

interface HandoffPillsProps {
  handoffs: Handoff[];
}

export const HandoffPills = ({ handoffs }: HandoffPillsProps) => (
  <div className="flex flex-wrap gap-y-1">
    {handoffs.map((handoff) => (
      <HandoffPill key={handoff.id} handoff={handoff} />
    ))}
  </div>
);
