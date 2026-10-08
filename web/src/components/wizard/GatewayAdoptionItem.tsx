import type { GatewayAdoption } from "@/models/Machine";

interface GatewayAdoptionItemProps {
  adoption: GatewayAdoption;
}

export const GatewayAdoptionItem = ({ adoption: g }: GatewayAdoptionItemProps) => (
  <div className="space-y-1 text-sm">
    <p className="font-medium wrap-anywhere">{g.name}</p>
    {g.error && <p className="text-xs text-destructive">Not adopted as a gateway: {g.error}</p>}
    {g.gateway_id && g.exposed.length === 0 && g.unmatched.length === 0 && (
      <p className="text-xs text-muted-foreground">Adopted as a gateway; no hostnames routed through it yet.</p>
    )}
    {g.exposed.length > 0 && (
      <ul
        aria-label={`Hostnames now on the canvas via ${g.name}`}
        className="space-y-0.5 font-mono text-xs wrap-anywhere text-muted-foreground"
      >
        {g.exposed.map((host) => (
          <li key={host}>{host}</li>
        ))}
      </ul>
    )}
    {g.unmatched.length > 0 && (
      <p className="text-xs break-words text-muted-foreground">
        Not linked, their target is not a container here: {g.unmatched.join(", ")}
      </p>
    )}
  </div>
);
