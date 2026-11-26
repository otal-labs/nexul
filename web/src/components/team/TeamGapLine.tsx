import { statusDotClass } from "@/components/settings/SettingsStatus";
import { cn } from "@/lib/utils";
import type { TeamPerson } from "@/models/Team";
import { gapText, skippedText, teamProgress } from "@/utils/TeamProgressUtility";

// One status mark while a Welcome wizard step is left or skipped: warning while one is left, muted once only skips
// remain, nothing when every step is done. Its words stay muted meta beside the handles; only the dot carries the tone.
export const TeamGapLine = ({ person }: { person: TeamPerson }) => {
  const progress = teamProgress(person);
  if (!progress || progress.left.length + progress.skipped.length === 0) return null;
  const { done, steps, left, skipped } = progress;
  const detail = left.length > 0 ? gapText(left) : skippedText(skipped);
  return (
    <span className="inline-flex shrink-0 items-center gap-1.5">
      <span aria-hidden className={cn("size-1.5 rounded-full", statusDotClass[left.length > 0 ? "warning" : "muted"])} />
      <span>
        <span className="tabular-nums">
          {done} of {steps.length}
        </span>{" "}
        set up<span className="sr-only">, {detail}</span>
      </span>
    </span>
  );
};
