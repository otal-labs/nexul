import { EnterList } from "@/components/EnterList";
import { ServiceCard } from "@/components/service/ServiceCard";
import { settingsListClass } from "@/components/settings/SettingsRow";
import type { ServiceDef } from "@/models/Service";

interface ServicesFeedProps {
  services: ServiceDef[];
}

// Hairline rows straight on the card, no column heads: each row names itself and the repository it builds from, and the machine only when they differ, since the header names a shared one.
export const ServicesFeed = ({ services }: ServicesFeedProps) => {
  const sharedMachine = new Set(services.map((svc) => svc.machine || svc.target)).size === 1;
  return (
    <EnterList className={settingsListClass}>
      {services.map((svc) => (
        <ServiceCard key={svc.id} service={svc} showMachine={!sharedMachine} />
      ))}
    </EnterList>
  );
};
