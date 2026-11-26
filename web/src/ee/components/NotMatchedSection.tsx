import { EmptyRow } from "@/components/EmptyRow";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { settingsListClass } from "@/components/settings/SettingsRow";
import { NotMatchedRow } from "@/ee/components/NotMatchedRow";
import { otherProtocolName, unmatchedPeople, type SignInMoves } from "@/ee/models/SignInMove";

// Everyone still on the other protocol that nothing matches yet.
export const NotMatchedSection = ({ moves }: { moves: SignInMoves }) => {
  const people = unmatchedPeople(moves);
  return (
    <SettingsCard
      id="not-matched"
      title="Not matched"
      description={`These people still sign in through ${otherProtocolName(moves.protocol)}, and no account a first sign-in through ${moves.provider_name} made has their email. Expect their sign-in so their first one lands on their account, or pick the account it already made.`}
    >
      {people.length === 0 && <EmptyRow flush>Everyone still on the other sign-in has a match or an expected sign-in.</EmptyRow>}
      {people.length > 0 && (
        <ul className={settingsListClass}>
          {people.map((person) => (
            <NotMatchedRow key={person.id} person={person} providerName={moves.provider_name} candidates={moves.unmatched} />
          ))}
        </ul>
      )}
    </SettingsCard>
  );
};
