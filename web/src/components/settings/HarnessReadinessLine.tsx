import { useHarnessReadiness, useListComputers } from "@/hooks/PairingHooks";

// The same "can this user run a play right now" line the ticket page's button states will read from.
export const HarnessReadinessLine = () => {
  const readiness = useHarnessReadiness();
  const { data: computers } = useListComputers();
  const computerName = readiness?.state === "ready" ? computers?.find((c) => c.id === readiness.computerId)?.name : undefined;

  return (
    <div>
      {readiness && readiness.state === "ready" && (
        <p className="text-sm text-success">Ready to run plays{computerName ? ` on ${computerName}` : ""}</p>
      )}
      {readiness && readiness.state !== "ready" && <p className="text-sm text-muted-foreground">{readiness.message}</p>}
    </div>
  );
};
