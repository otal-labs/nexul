import { Link2 } from "lucide-react";
import { useState } from "react";

import { ActionButton } from "@/components/action/ActionButton";
import { EmptyRow } from "@/components/EmptyRow";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { settingsListClass } from "@/components/settings/SettingsRow";
import { LikelyDuplicateRow } from "@/ee/components/LikelyDuplicateRow";
import { useLinkMatched } from "@/ee/hooks/SignInMoveHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { matchedPeople, otherProtocolName, type MoveAccount, type OldSignIn, type SignInMoves } from "@/ee/models/SignInMove";

type Matched = OldSignIn & { match: MoveAccount };

const isMatched = (person: OldSignIn): person is Matched => person.match !== null;

const pairsWord = (count: number): string => (count === 1 ? "1 sign-in" : `${count} sign-ins`);

// The pairs whose email matches: ticked by default, linked together by one Link, each moving on its own.
export const LikelyDuplicatesSection = ({ moves }: { moves: SignInMoves }) => {
  const [skipped, setSkipped] = useState<ReadonlySet<string>>(new Set());
  const link = useLinkMatched();
  const { open: confirm } = useConfirmationDialog();
  const matched = matchedPeople(moves).filter(isMatched);
  const chosen = matched.filter((person) => !skipped.has(person.id));

  const toggle = (id: string, checked: boolean) =>
    setSkipped((current) => {
      const next = new Set(current);
      if (checked) next.delete(id);
      if (!checked) next.add(id);
      return next;
    });

  const onLink = async () => {
    const ok = await confirm({
      title: `Link ${pairsWord(chosen.length)}?`,
      message: `Each person signs in through ${moves.provider_name} to their own account from now on. The accounts the first sign-ins made are disabled and signed out, and what they made stays.`,
      confirmLabel: "Link",
      destructive: false,
    });
    if (ok) link.mutate(chosen.map((person) => ({ from_account_id: person.match.id, to_account_id: person.id })));
  };

  return (
    <SettingsCard
      id="likely-duplicates"
      title="Likely duplicates"
      description={`These people still sign in through ${otherProtocolName(moves.protocol)}, and a first sign-in through ${moves.provider_name} made a second account with the same email. Linking moves that sign-in onto their own account and disables the second one.`}
      footer={
        matched.length > 0 && (
          <ActionButton
            permission="sign_ins:move"
            verb="link sign-ins"
            className="ml-auto"
            size="sm"
            icon={Link2}
            disabled={chosen.length === 0}
            loading={link.isPending}
            onClick={() => void onLink()}
          >
            Link {pairsWord(chosen.length)}
          </ActionButton>
        )
      }
    >
      {matched.length === 0 && <EmptyRow flush>No account a first sign-in made matches anyone still on the other sign-in.</EmptyRow>}
      {matched.length > 0 && (
        <ul className={settingsListClass}>
          {matched.map((person) => (
            <LikelyDuplicateRow key={person.id} person={person} checked={!skipped.has(person.id)} onCheckedChange={(checked) => toggle(person.id, checked)} />
          ))}
        </ul>
      )}
    </SettingsCard>
  );
};
