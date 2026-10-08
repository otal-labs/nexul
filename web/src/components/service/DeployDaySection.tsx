import { Microheader } from "@/components/Microheader";
import { DeployRow } from "@/components/service/DeployRow";
import type { Deploy } from "@/models/Stack";

interface DeployDaySectionProps {
  label: string;
  deploys: Deploy[];
}

export const DeployDaySection = ({ label, deploys }: DeployDaySectionProps) => (
  <li>
    <Microheader className="px-3 pt-3 pb-1">{label}</Microheader>
    <ul className="divide-y divide-border">
      {deploys.map((deploy) => (
        <DeployRow key={deploy.id} deploy={deploy} />
      ))}
    </ul>
  </li>
);
