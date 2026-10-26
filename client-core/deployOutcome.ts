// A deploy's state in deploy words: "healthy" is how a service is doing, and beside the Health fact it contradicts it.
const OUTCOME: Record<string, string> = {
  pending: "Pending",
  running: "Running",
  healthy: "Succeeded",
  failed: "Failed",
};

export const deployOutcome = (status: string): string => OUTCOME[status] ?? status;
