import { avatarFill } from "@nexul/client-core/avatarFill";

import type { CollabParticipant } from "@/components/doc/collab/useCollabSession";
import { usePerson } from "@/hooks/PeopleHooks";
import { cn } from "@/lib/utils";

interface PresenceSectionProps {
  participants: CollabParticipant[];
  max?: number;
  /** "sm" sits beside 12px meta text: a 20px avatar instead of 24px. */
  size?: "sm" | "md";
}

const sizes = { sm: "size-5 text-[10px]", md: "size-6 text-xs" };

// Stacked avatar group for glanceable presence; no domain event per cursor move.
export const PresenceSection = ({ participants, max = 4, size = "md" }: PresenceSectionProps) => {
  if (participants.length === 0) return null;
  const shown = participants.slice(0, max);
  const overflow = participants.length - shown.length;

  return (
    <div className="flex items-center -space-x-1" data-testid="presence-section">
      {shown.map((p) => (
        <PresenceAvatar key={p.clientID} participant={p} sizeClass={sizes[size]} />
      ))}
      {overflow > 0 && (
        <span
          aria-label={`${overflow} more`}
          className={cn("avatar-more ring-2 ring-background", sizes[size])}
        >
          +{overflow}
        </span>
      )}
    </div>
  );
};

// A known account takes the fill its avatar has everywhere else; the session's colour stays for its cursor.
const PresenceAvatar = ({ participant: p, sizeClass }: { participant: CollabParticipant; sizeClass: string }) => {
  const person = usePerson(p.id ?? "");
  const fill = p.id ? avatarFill(person.login) : p.color;
  return (
    <span
      title={`${p.name} (${p.activity})`}
      aria-label={`${p.name} ${p.activity}`}
      className={cn(
        "flex items-center justify-center overflow-hidden rounded-full font-semibold text-white shadow-card ring-2 ring-background transition-transform duration-150 ease-standard hover:z-10 hover:-translate-y-0.5 motion-reduce:hover:translate-y-0",
        sizeClass,
      )}
      style={{ backgroundColor: fill }}
      data-testid="presence-avatar"
    >
      {p.avatar && <img src={p.avatar} alt="" referrerPolicy="no-referrer" className="size-full object-cover" />}
      {!p.avatar && p.name.charAt(0).toUpperCase()}
    </span>
  );
};
