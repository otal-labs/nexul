import { SettingsStatus } from "@/components/settings/SettingsStatus";
import type { TeamPerson } from "@/models/Team";
import { gapText, skippedText, teamProgress } from "@/utils/TeamProgressUtility";

// One status mark while a Welcome wizard step is left or skipped: warning while one is left, muted once only skips
// remain, nothing when every step is done. What is left is read aloud here and shown in the person's dialog.
export const TeamGapLine = ({ person }: { person: TeamPerson }) => {
  const progress = teamProgress(person);
  if (!progress || progress.left.length + progress.skipped.length === 0) return null;
  const { done, steps, left, skipped } = progress;
  const detail = left.length > 0 ? gapText(left) : skippedText(skipped);
  return (
    <SettingsStatus tone={left.length > 0 ? "warning" : "muted"}>
      <span className="tabular-nums">
        {done} of {steps.length}
      </span>{" "}
      set up<span className="sr-only">, {detail}</span>
    </SettingsStatus>
  );
};
