import { UserMinus, UserRoundCheck, UserRoundX } from "lucide-react";

import { ActionButton } from "@/components/action/ActionButton";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useRemoveAccount, useUpdateAccountStatus } from "@/hooks/TeamHooks";
import type { TeamPerson } from "@/models/Team";
import { personName } from "@/utils/TeamUtility";

// Account status is instance-wide: disabling blocks sign-in everywhere, and each way in has its way back. Changing it
// takes accounts:write and removing an account accounts:delete, each held in any workspace.
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
      message: `${name} loses sign-in, their credentials, and access to ${workspaces} workspace${workspaces === 1 ? "" : "s"}. Their computers are disconnected. What they wrote stays. Restoring the account later doesn't bring the access or the computers back.`,
      confirmLabel: "Remove account",
    });
    if (ok) remove.mutate(person.id);
  };

  return (
    <div className="flex flex-wrap items-center gap-1">
      {person.status === "active" && (
        <ActionButton permission="accounts:write" verb="disable accounts" icon={UserRoundX} type="button" variant="ghost" size="sm" className="text-muted-foreground" liveClassName="hover:text-destructive" loading={update.isPending} disabled={busy} onClick={() => void setStatus("disabled", "Disable", "lose sign-in but keep their workspace access, and their computers are disconnected")}>
          Disable
        </ActionButton>
      )}
      {person.status === "disabled" && (
        <ActionButton permission="accounts:write" verb="reactivate accounts" icon={UserRoundCheck} type="button" variant="ghost" size="sm" loading={update.isPending} disabled={busy} onClick={() => void setStatus("active", "Reactivate", "be able to sign in again. The computers disconnected when the account was disabled stay disconnected")}>
          Reactivate
        </ActionButton>
      )}
      {person.status === "removed" && (
        <ActionButton permission="accounts:write" verb="restore accounts" icon={UserRoundCheck} type="button" variant="ghost" size="sm" loading={update.isPending} disabled={busy} onClick={() => void setStatus("active", "Restore", "be able to sign in again, with no workspace access until you add it")}>
          Restore
        </ActionButton>
      )}
      {person.status !== "removed" && (
        <ActionButton permission="accounts:delete" verb="remove accounts" icon={UserMinus} type="button" variant="ghost" size="sm" className="text-muted-foreground" liveClassName="hover:text-destructive" loading={remove.isPending} disabled={busy} onClick={() => void removeAccount()}>
          Remove account
        </ActionButton>
      )}
    </div>
  );
};
