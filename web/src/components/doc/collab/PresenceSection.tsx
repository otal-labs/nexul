import type { CollabParticipant } from "@/components/doc/collab/useCollabSession";

interface PresenceSectionProps {
  participants: CollabParticipant[];
  max?: number;
}

// Stacked avatar group for glanceable presence; no domain event per cursor move.
export const PresenceSection = ({ participants, max = 4 }: PresenceSectionProps) => {
  if (participants.length === 0) return null;
  const shown = participants.slice(0, max);
  const overflow = participants.length - shown.length;

  return (
    <div className="flex items-center -space-x-2" data-testid="presence-section">
      {shown.map((p) => (
        <span
          key={p.clientID}
          title={`${p.name} (${p.activity})`}
          aria-label={`${p.name} ${p.activity}`}
          className="flex size-7 items-center justify-center overflow-hidden rounded-full text-xs font-semibold text-white shadow-card ring-2 ring-background transition-transform duration-150 ease-standard hover:z-10 hover:-translate-y-0.5 motion-reduce:hover:translate-y-0"
          style={{ backgroundColor: p.color }}
          data-testid="presence-avatar"
        >
          {p.avatar && (
            <img src={p.avatar} alt="" referrerPolicy="no-referrer" className="size-full object-cover" />
          )}
          {!p.avatar && p.name.charAt(0).toUpperCase()}
        </span>
      ))}
      {overflow > 0 && (
        <span
          aria-label={`${overflow} more`}
          className="flex size-7 items-center justify-center rounded-full bg-muted text-xs font-semibold text-muted-foreground ring-2 ring-background"
        >
          +{overflow}
        </span>
      )}
    </div>
  );
};
