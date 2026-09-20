import { UserRoundCheck, UserRoundX } from "lucide-react";

import { ConfirmDestroyButton } from "@/components/settings/ConfirmDestroyButton";
import { Button } from "@/components/ui/button";
import type { Account } from "@/models/Invitation";

interface AccountRowProps {
  account: Account;
  busy: boolean;
  onToggle: (account: Account) => void;
  onRemove: (account: Account) => void;
}

export const AccountRow = ({ account, busy, onToggle, onRemove }: AccountRowProps) => (
  <li className="flex flex-wrap items-center gap-3 px-3 py-3 transition-colors duration-150 ease-standard hover:bg-accent/40 sm:px-4">
    <span className="flex size-7 shrink-0 items-center justify-center rounded-full bg-muted text-xs font-medium">{account.login.charAt(0).toUpperCase()}</span>
    <div className="min-w-0 flex-1"><p className="truncate font-medium">{account.name || account.login}</p><p className="truncate font-mono text-xs text-muted-foreground">@{account.login}</p></div>
    <span className="font-mono text-[11px] uppercase tracking-wide text-muted-foreground">{account.status}</span>
    {account.status === "removed" && <Button type="button" variant="outline" size="sm" disabled={busy} onClick={() => onToggle(account)}><UserRoundCheck className="size-4" />Restore</Button>}
    {account.status === "disabled" && <Button type="button" variant="outline" size="sm" disabled={busy} onClick={() => onToggle(account)}><UserRoundCheck className="size-4" />Reactivate</Button>}
    {account.status === "active" && <Button type="button" variant="ghost" size="sm" disabled={busy} onClick={() => onToggle(account)}><UserRoundX className="size-4" />Disable</Button>}
    {account.status !== "removed" && <ConfirmDestroyButton icon={UserRoundX} idleLabel="Remove account" onConfirm={() => onRemove(account)} disabled={busy} />}
  </li>
);
