import { EnterList } from "@/components/EnterList";
import { ServiceCard } from "@/components/service/ServiceCard";
import { settingsRowListClass } from "@/components/settings/SettingsRow";
import type { ServiceDef } from "@/models/Service";

interface ServicesFeedProps {
  services: ServiceDef[];
}

// Hairline rows straight on the card, no column heads: each row names itself and says where it runs.
export const ServicesFeed = ({ services }: ServicesFeedProps) => (
  <EnterList className={settingsRowListClass}>
    {services.map((svc) => (
      <ServiceCard key={svc.id} service={svc} />
    ))}
  </EnterList>
);
