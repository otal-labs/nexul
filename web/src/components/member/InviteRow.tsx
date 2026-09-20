import { UserX } from "lucide-react";

import { Button } from "@/components/ui/button";

interface InviteRowProps {
  login: string;
  roleName: string;
  isCancelling: boolean;
  onCancel: (login: string) => void;
  index: number;
}

const statusChipClass =
  "inline-flex shrink-0 items-center rounded-full bg-warning/15 px-2 py-0.5 font-mono text-[11px] font-medium uppercase tracking-wide text-warning";

// Pending invite, not role-editable: cancel and re-invite with a different role instead.
export const InviteRow = ({ login, roleName, isCancelling, onCancel, index }: InviteRowProps) => {
  const enterDelayMs = Math.min(index, 7) * 25;

  return (
    <li className="transition-colors duration-150 ease-standard hover:bg-accent/40">
      <div
        className="animate-in fade-in-0 slide-in-from-bottom-1 flex items-center gap-2 px-3 py-3 duration-150 ease-out sm:gap-3 sm:px-4"
        style={{ animationDelay: `${enterDelayMs}ms` }}
      >
        <span
          aria-hidden
          className="flex size-6 shrink-0 items-center justify-center rounded-full bg-accent text-xs font-medium text-accent-foreground"
        >
          {login.charAt(0).toUpperCase()}
        </span>
        <div className="min-w-0 flex-1">
          <div className="flex items-baseline gap-2">
            <span className="truncate font-medium">{login}</span>
            <span className={statusChipClass}>Invite sent</span>
          </div>
          <p className="truncate font-mono text-xs text-muted-foreground">{roleName}</p>
        </div>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          aria-label={`Withdraw invite for ${login}`}
          title="Withdraw invite"
          disabled={isCancelling}
          onClick={() => onCancel(login)}
        >
          <UserX className="size-4" />
        </Button>
      </div>
    </li>
  );
};
