import { SettingsStatus } from "@/components/settings/SettingsStatus";
import type { TeamPerson } from "@/models/Team";
import { gapText, skippedText, teamProgress } from "@/utils/TeamProgressUtility";

// The line under a person while a Welcome wizard step is left or skipped: warning while one is left, muted once only
// skips remain, and nothing when every step is done.
export const TeamGapLine = ({ person }: { person: TeamPerson }) => {
  const progress = teamProgress(person);
  if (!progress || progress.left.length + progress.skipped.length === 0) return null;
  const { done, steps, left, skipped } = progress;
  const count = (
    <>
      <span className="tabular-nums">
        {done} of {steps.length}
      </span>{" "}
      set up
    </>
  );
  return (
    <span className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-0.5">
      {left.length > 0 && (
        <SettingsStatus tone="warning" detail={gapText(left)}>
          {count}
        </SettingsStatus>
      )}
      {left.length === 0 && <SettingsStatus tone="muted">{count}</SettingsStatus>}
      {skipped.length > 0 && <span className="text-xs text-muted-foreground">· {skippedText(skipped)}</span>}
    </span>
  );
};
