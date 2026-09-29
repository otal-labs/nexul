import { cn } from "@/lib/utils";
import type { AccountStatus } from "@/models/Team";

const dotClass: Record<AccountStatus, string> = {
  active: "bg-success",
  disabled: "bg-warning",
  removed: "bg-destructive",
};

export const AccountStatusLabel = ({ status }: { status: AccountStatus }) => (
  <span className="inline-flex shrink-0 items-center gap-1.5 font-mono text-[11px] uppercase tracking-wide text-muted-foreground">
    <span aria-hidden className={cn("size-1.5 rounded-full", dotClass[status])} />
    {status}
  </span>
);
