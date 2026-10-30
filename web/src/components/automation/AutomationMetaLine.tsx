import { AutomationKindBadge } from "@/components/automation/AutomationKindBadge";
import { RunnerStatusDot } from "@/components/runner/RunnerStatusDot";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useFetchAutomationHosts } from "@/hooks/AutomationHostHooks";
import type { Automation } from "@/models/Automation";

interface AutomationMetaLineProps {
  automation: Automation;
}

// What it is, where it runs, what it hears and which token it signs in with; the instance host has no dot, it is never offline.
export const AutomationMetaLine = ({ automation }: AutomationMetaLineProps) => {
  const { data: hosts } = useFetchAutomationHosts();
  const host = automation.host_id ? hosts?.find((h) => h.id === automation.host_id) : undefined;
  const events = automation.subscriptions.length;

  return (
    <>
      <AutomationKindBadge kind={automation.kind} />
      <span className="flex items-center gap-1.5 text-xs">
        {host && <RunnerStatusDot connected={host.connected} />}
        Runs on {automation.host_id ? (host?.name ?? "its host") : "the instance host"}
      </span>
      {events > 0 && (
        <Tooltip>
          <TooltipTrigger asChild>
            <span tabIndex={0} className="text-xs tabular-nums">
              {events} {events === 1 ? "event" : "events"}
            </span>
          </TooltipTrigger>
          <TooltipContent className="font-mono">
            {automation.subscriptions.map((topic) => (
              <p key={topic}>{topic}</p>
            ))}
          </TooltipContent>
        </Tooltip>
      )}
      {automation.token_prefix && <span className="font-mono text-xs">dat_…{automation.token_prefix}</span>}
    </>
  );
};
