import { useHarnessReadiness, useListComputers } from "@/hooks/PairingHooks";
import { passOverLines } from "@/models/Pairing";
import { cn } from "@/lib/utils";

const CHATS = "Chats outside a project";

// Where a chat outside any project would start right now; a project's runs are each computer's own line below.
export const HarnessReadinessLine = () => {
  const readiness = useHarnessReadiness();
  const { data: computers } = useListComputers();
  if (!readiness) return null;

  const ready = readiness.state === "ready";
  const computerName = ready ? computers?.find((c) => c.id === readiness.computerId)?.name : undefined;
  const passes = readiness.state === "no_computer_ready" ? (readiness.passedOver ?? []) : [];
  return (
    <p className="flex items-start gap-1.5 text-xs">
      <span aria-hidden className={cn("mt-[5px] size-1.5 shrink-0 rounded-full", ready ? "bg-success" : "bg-warning")} />
      {ready && `${CHATS} start on ${computerName ?? "your first computer"}.`}
      {readiness.state === "no_computer_ready" && (
        <span className="text-muted-foreground">
          {CHATS} can't start{passes.length > 0 && `: ${passOverLines(passes)}`}.
        </span>
      )}
      {readiness.state !== "ready" && readiness.state !== "no_computer_ready" && <span className="text-muted-foreground">{readiness.message}</span>}
    </p>
  );
};
