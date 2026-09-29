import { UserMinus, UserRoundCheck, UserRoundX } from "lucide-react";

import { Button } from "@/components/ui/button";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useRemoveAccount, useUpdateAccountStatus } from "@/hooks/TeamHooks";
import type { TeamPerson } from "@/models/Team";
import { personName } from "@/utils/TeamUtility";

// Account status is instance-wide: disabling blocks sign-in everywhere, and each way in has its way back.
export const TeamAccountActions = ({ person }: { person: TeamPerson }) => {
  const update = useUpdateAccountStatus();
  const remove = useRemoveAccount();
  const { open: confirm } = useConfirmationDialog();
  const busy = update.isPending || remove.isPending;
  const name = personName(person);

  const setStatus = async (status: "active" | "disabled", action: string, consequence: string) => {
    const ok = await confirm({ title: `${action} account?`, message: `${name} will ${consequence}.`, confirmLabel: action, destructive: status === "disabled" });
    if (ok) update.mutate({ id: person.id, status });
  };

  const removeAccount = async () => {
    const workspaces = person.workspaces.length;
    const ok = await confirm({
      title: "Remove account?",
      message: `${name} will lose sign-in, their credentials, and their access to ${workspaces} workspace${workspaces === 1 ? "" : "s"}. What they wrote stays. Restoring the account later does not bring the access back.`,
      confirmLabel: "Remove account",
    });
    if (ok) remove.mutate(person.id);
  };

  return (
    <div className="flex flex-wrap items-center gap-1">
      {person.status === "active" && (
        <Button type="button" variant="ghost" size="sm" loading={update.isPending} disabled={busy} onClick={() => void setStatus("disabled", "Disable", "no longer be able to sign in; their workspace access is kept")}>
          <UserRoundX className="size-4" />Disable
        </Button>
      )}
      {person.status === "disabled" && (
        <Button type="button" variant="ghost" size="sm" loading={update.isPending} disabled={busy} onClick={() => void setStatus("active", "Reactivate", "be able to sign in again")}>
          <UserRoundCheck className="size-4" />Reactivate
        </Button>
      )}
      {person.status === "removed" && (
        <Button type="button" variant="ghost" size="sm" loading={update.isPending} disabled={busy} onClick={() => void setStatus("active", "Restore", "be able to sign in again, with no workspace access until you add it")}>
          <UserRoundCheck className="size-4" />Restore
        </Button>
      )}
      {person.status !== "removed" && (
        <Button type="button" variant="ghost" size="sm" className="hover:text-destructive" loading={remove.isPending} disabled={busy} onClick={() => void removeAccount()}>
          <UserMinus className="size-4" />Remove account
        </Button>
      )}
    </div>
  );
};
