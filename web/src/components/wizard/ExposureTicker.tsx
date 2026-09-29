import { errorMessage } from "@/api/client";
import { TickerRow, type CheckOutcome } from "@/components/TickerRow";

const STAGES = [
  { label: "Gateway", why: "Reuses a gateway on the service's machine, or starts one." },
  { label: "Docker network", why: "Connects the gateway to the network the service runs on." },
  { label: "DNS record", why: "Points the hostname at the gateway: a tunnel route and CNAME, or an A record." },
  { label: "Saved", why: "Records the exposure so the stack and the canvas show it." },
];

interface ExposureTickerProps {
  status: "idle" | "pending" | "success" | "error";
  error: unknown;
  result: string | undefined;
}

// The exposure endpoint answers once, after every stage: the rows wait together, then all resolve from that one response.
export const ExposureTicker = ({ status, error, result }: ExposureTickerProps) => (
  <div className="space-y-3">
    {status !== "idle" && (
      <ul className="space-y-2" aria-label="Exposure progress">
        {STAGES.map((stage, index) => (
          <TickerRow key={stage.label} label={stage.label} why={stage.why} outcome={outcomeFor(status, index === STAGES.length - 1 ? result : undefined)} />
        ))}
      </ul>
    )}
    {status === "error" && (
      <p role="alert" className="text-sm text-destructive">
        {errorMessage(error)}
      </p>
    )}
  </div>
);

const outcomeFor = (status: ExposureTickerProps["status"], result: string | undefined): CheckOutcome => {
  if (status === "pending") return { state: "pending" };
  if (status === "success") return result ? { state: "ok", message: result } : { state: "ok" };
  return { state: "idle" };
};
