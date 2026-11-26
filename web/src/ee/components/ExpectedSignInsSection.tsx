import { EmptyRow } from "@/components/EmptyRow";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { settingsListClass } from "@/components/settings/SettingsRow";
import { ExpectedSignInRow } from "@/ee/components/ExpectedSignInRow";
import type { SignInMoves } from "@/ee/models/SignInMove";

// The expected sign-ins still waiting for their first sign-in, each of which beats the match by email.
export const ExpectedSignInsSection = ({ moves }: { moves: SignInMoves }) => (
  <SettingsCard
    id="expected-sign-ins"
    title="Expected sign-ins"
    description={`The first sign-in through ${moves.provider_name || "the provider"} with one of these emails opens the account beside it, and the expected sign-in goes.`}
  >
    {moves.expected.length === 0 && <EmptyRow flush>No sign-in is expected.</EmptyRow>}
    {moves.expected.length > 0 && (
      <ul className={settingsListClass}>
        {moves.expected.map((expected) => (
          <ExpectedSignInRow key={expected.account.id} expected={expected} />
        ))}
      </ul>
    )}
  </SettingsCard>
);
