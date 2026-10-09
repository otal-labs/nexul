import { EnterList } from "@/components/EnterList";
import { ServiceCard } from "@/components/service/ServiceCard";
import type { ServiceDef } from "@/models/Service";

interface ServicesFeedProps {
  services: ServiceDef[];
}

// Hairline rows in one bordered box, no column heads: each row names itself and says where it comes from.
export const ServicesFeed = ({ services }: ServicesFeedProps) => (
  <EnterList className="divide-y divide-border overflow-hidden rounded-md border border-border">
    {services.map((svc) => (
      <ServiceCard key={svc.id} service={svc} />
    ))}
  </EnterList>
);
