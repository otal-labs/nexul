import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { cn } from "@/lib/utils";
import { LEFT_DETAIL, TEAM_STEP_NAMES, type TeamStep } from "@/utils/TeamProgressUtility";
import { formatRelativeTime } from "@/utils/TimeUtility";

// One step left or skipped: its name, why from the data, and where it stands.
export const TeamWelcomeStepRow = ({ step }: { step: TeamStep }) => (
  <li className="flex items-start gap-3 px-3 py-2.5 sm:px-4">
    <div className="min-w-0 flex-1">
      <p className={cn("text-sm font-medium", step.state === "skipped" && "text-muted-foreground")}>{TEAM_STEP_NAMES[step.key]}</p>
      <p className="text-xs text-muted-foreground">
        {step.state === "left" && LEFT_DETAIL[step.key]}
        {step.state === "skipped" && "They chose to skip it; it comes back if a role gives it again"}
      </p>
    </div>
    <span className="shrink-0 pt-0.5">
      {step.state === "left" && <SettingsStatus tone="warning">Not done</SettingsStatus>}
      {step.state === "skipped" && (
        <SettingsStatus tone="muted" detail={step.at && formatRelativeTime(step.at)}>
          Skipped
        </SettingsStatus>
      )}
    </span>
  </li>
);
