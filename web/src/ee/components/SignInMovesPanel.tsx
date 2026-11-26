import { ErrorDisplay } from "@/components/ErrorDisplay";
import { EmptyRow } from "@/components/EmptyRow";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { ExpectedSignInsSection } from "@/ee/components/ExpectedSignInsSection";
import { LikelyDuplicatesSection } from "@/ee/components/LikelyDuplicatesSection";
import { NotMatchedSection } from "@/ee/components/NotMatchedSection";
import { useSignInMoves } from "@/ee/hooks/SignInMoveHooks";

// Settings, Sign-in providers, Move sign-ins: who still signs in through the other protocol, and the ways to move them.
export const SignInMovesPanel = () => {
  const { data: moves, error, isPending } = useSignInMoves();
  return (
    <>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {moves && moves.protocol === "" && (
        <SettingsCard id="likely-duplicates" title="No provider saved">
          <EmptyRow flush>Set up the single sign-on provider people move to first; then this page lists who still signs in through the other one.</EmptyRow>
        </SettingsCard>
      )}
      {moves && moves.protocol !== "" && <LikelyDuplicatesSection moves={moves} />}
      {moves && moves.protocol !== "" && <NotMatchedSection moves={moves} />}
      {moves && <ExpectedSignInsSection moves={moves} />}
    </>
  );
};
