import { cn } from "@/lib/utils";
import type { AccountStatus } from "@/models/Team";

const labelClass: Record<AccountStatus, string> = {
  active: "text-muted-foreground",
  disabled: "text-warning",
  removed: "text-destructive",
};

interface AccountStatusLabelProps {
  status: AccountStatus;
  online: boolean;
}

// The dot is presence, the word is the account status: disabled and removed accounts keep their status colour on the word.
export const AccountStatusLabel = ({ status, online }: AccountStatusLabelProps) => (
  <span className={cn("inline-flex shrink-0 items-center gap-1.5 font-mono text-[11px] uppercase tracking-wide", labelClass[status])}>
    <span aria-hidden className={cn("size-1.5 rounded-full", online ? "bg-success" : "bg-muted-foreground")} />
    {status}
  </span>
);
