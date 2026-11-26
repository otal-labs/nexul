import { ChevronDown, MailCheck } from "lucide-react";

import { ActionButton } from "@/components/action/ActionButton";
import { ActionMenuItem } from "@/components/action/ActionMenuItem";
import { dropdownParts } from "@/components/listpane/rowMenu";
import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuLabel, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { ExpectSignInForm } from "@/ee/components/ExpectSignInForm";
import { SignInMovePerson } from "@/ee/components/SignInMovePerson";
import { useLinkMatched } from "@/ee/hooks/SignInMoveHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useFormDialog } from "@/hooks/useFormDialog";
import { ExpectSignInFormSchema, type ExpectSignInFormData, type MoveAccount, type OldSignIn } from "@/ee/models/SignInMove";

interface NotMatchedRowProps {
  person: OldSignIn;
  providerName: string;
  // The accounts a first sign-in made that match nobody, any of which may be this person's.
  candidates: MoveAccount[];
}

// Someone with no suggestion: expect their first sign-in, or pick the account a first sign-in already made for them.
export const NotMatchedRow = ({ person, providerName, candidates }: NotMatchedRowProps) => {
  const { open: openExpect } = useFormDialog();
  const { open: confirm } = useConfirmationDialog();
  const link = useLinkMatched();
  const name = person.name || person.login;

  const onExpect = () =>
    void openExpect<ExpectSignInFormData>({
      title: `Expect ${name}'s sign-in`,
      schema: ExpectSignInFormSchema,
      okLabel: "Expect sign-in",
      form: <ExpectSignInForm accountId={person.id} providerName={providerName} />,
      formOptions: { defaultValues: { email: person.email } },
    });

  const onPick = async (account: MoveAccount) => {
    const ok = await confirm({
      title: `Link ${account.email} to ${name}?`,
      message: `Signing in through ${providerName} as ${account.email} opens ${name}'s account from now on. The account that sign-in made is disabled and signed out, and what it made stays.`,
      confirmLabel: "Link",
      destructive: false,
    });
    if (ok) link.mutate([{ from_account_id: account.id, to_account_id: person.id }]);
  };

  return (
    <li className="flex flex-wrap items-center gap-3 py-3">
      <SignInMovePerson account={person} className="flex-1" />
      <span className="flex shrink-0 gap-2">
        <ActionButton permission="sign_ins:move" verb="expect a sign-in" variant="outline" size="sm" icon={MailCheck} onClick={onExpect}>
          Expect sign-in
        </ActionButton>
        {candidates.length > 0 && (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button type="button" variant="ghost" size="sm" loading={link.isPending}>
                Pick account
                <ChevronDown aria-hidden />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="min-w-64">
              <DropdownMenuLabel>Made by a first sign-in</DropdownMenuLabel>
              {candidates.map((account) => (
                <ActionMenuItem key={account.id} permission="sign_ins:move" verb="link sign-ins" ui={dropdownParts} onSelect={() => void onPick(account)}>
                  <span className="min-w-0 truncate">{account.name || account.login}</span>
                  <span className="ml-auto truncate font-mono text-xs text-muted-foreground">{account.email}</span>
                </ActionMenuItem>
              ))}
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </span>
    </li>
  );
};
