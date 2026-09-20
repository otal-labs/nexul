import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useFetchAccounts, useRemoveAccount, useUpdateAccountStatus } from "@/hooks/InvitationHooks";
import { AccountRow } from "@/components/settings/AccountRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { SettingsCard } from "@/components/settings/SettingsCard";

export const AccountsSection = () => {
  const { data: accounts, isPending, error } = useFetchAccounts();
  const update = useUpdateAccountStatus();
  const remove = useRemoveAccount();
  const { open: confirm } = useConfirmationDialog();

  const toggle = async (account: Parameters<typeof AccountRow>[0]["account"]) => {
    const next = account.status === "active" ? "disabled" : "active";
    const ok = await confirm({ title: `${next === "disabled" ? "Disable" : "Reactivate"} account?`, message: `${account.name || account.login} will ${next === "disabled" ? "no longer be able to sign in" : "be able to sign in again"}.`, destructive: next === "disabled" });
    if (ok) update.mutate({ id: account.id, status: next });
  };

  const removeAccount = async (account: Parameters<typeof AccountRow>[0]["account"]) => {
    const ok = await confirm({ title: "Remove account?", message: `${account.name || account.login} will lose memberships and credentials. Their authored content remains.`, confirmLabel: "Remove account" });
    if (ok) remove.mutate(account.id);
  };

  return <SettingsCard id="instance-accounts" title="Registered accounts" description="Control which registered identities can access this private instance. New people enter through invitation links.">
    {isPending && <LoadingDisplay />}
    {error && <ErrorDisplay error={error} />}
    {accounts && accounts.length === 0 && <NoDataDisplay message="No registered accounts yet" />}
    {accounts && accounts.length > 0 && <ul className="divide-y divide-border overflow-hidden rounded-md border">{accounts.map((account) => <AccountRow key={account.id} account={account} busy={update.isPending || remove.isPending} onToggle={(value) => void toggle(value)} onRemove={(value) => void removeAccount(value)} />)}</ul>}
  </SettingsCard>;
};
