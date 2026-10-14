import { Button } from "@/components/ui/button";
import { useRevokeEnrollment } from "@/hooks/RunnerHooks";
import type { OpenEnrollment } from "@/models/Runner";
import { formatClockTime } from "@/utils/TimeUtility";

interface OpenEnrollmentRowProps {
  code: OpenEnrollment;
  /** What the code adds: the runner's name, or the computer's. */
  label: string;
}

export const OpenEnrollmentRow = ({ code, label }: OpenEnrollmentRowProps) => {
  const revoke = useRevokeEnrollment();
  return (
    <li className="flex items-center gap-3 px-3 py-2.5 transition-colors duration-150 ease-standard hover:bg-accent/40">
      <div className="min-w-0 flex-1">
        <span className="block truncate font-mono text-sm" title={label}>
          {label}
        </span>
        <span className="block truncate text-xs text-muted-foreground">
          {code.machine && `On ${code.machine}. `}
          Works once, until <span className="font-mono tabular-nums">{formatClockTime(code.expires_at)}</span>
        </span>
      </div>
      <Button type="button" variant="ghost" size="sm" aria-label={`Revoke the code for ${label}`} loading={revoke.isPending} onClick={() => revoke.mutate(code.id)}>
        Revoke
      </Button>
    </li>
  );
};
