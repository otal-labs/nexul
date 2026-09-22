import { DeployRow } from "@/components/service/DeployRow";
import type { Deploy } from "@/models/Stack";

interface DeployDaySectionProps {
  label: string;
  deploys: Deploy[];
}

export const DeployDaySection = ({ label, deploys }: DeployDaySectionProps) => (
  <div>
    <h3 className="mb-2 font-mono text-[11px] font-medium tracking-[0.18em] text-muted-foreground uppercase">
      {label}
    </h3>
    <ul className="divide-y divide-border overflow-hidden rounded-lg border border-border">
      {deploys.map((deploy) => (
        <DeployRow key={deploy.id} deploy={deploy} />
      ))}
    </ul>
  </div>
);
