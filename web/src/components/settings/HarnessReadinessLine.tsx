import { useHarnessReadiness, useListComputers } from "@/hooks/PairingHooks";
import { cn } from "@/lib/utils";

// Where a chat outside any project would go right now, read the way the ticket page's buttons read readiness.
export const HarnessReadinessLine = () => {
  const readiness = useHarnessReadiness();
  const { data: computers } = useListComputers();
  const computerName = readiness?.state === "ready" ? computers?.find((c) => c.id === readiness.computerId)?.name : undefined;

  return (
    <div>
      {readiness && (
        <p className="flex items-start gap-2 text-sm">
          <span
            aria-hidden
            className={cn("mt-1.5 size-1.5 shrink-0 rounded-full", readiness.state === "ready" ? "bg-success" : "bg-warning")}
          />
          {readiness.state === "ready" && `Ready: @Agent starts on ${computerName ?? "your first computer"}`}
          {readiness.state !== "ready" && <span className="text-muted-foreground">{readiness.message}</span>}
        </p>
      )}
    </div>
  );
};
