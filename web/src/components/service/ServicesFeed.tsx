import { ServiceCard } from "@/components/service/ServiceCard";
import type { ServiceDef } from "@/models/Service";

interface ServicesFeedProps {
  services: ServiceDef[];
}

// A column-header row over hairline body rows, not a card; cards are reserved for draggable units like TicketCard.
export const ServicesFeed = ({ services }: ServicesFeedProps) => (
  <div className="mt-2">
    <div className="flex items-center gap-3 border-b border-border px-4 pb-1.5 text-xs text-muted-foreground">
      <span className="flex-1">Name</span>
      <span className="hidden w-48 shrink-0 sm:inline">Source</span>
      <span className="w-20 shrink-0 text-right sm:inline">Strategy</span>
    </div>
    <ul className="divide-y divide-border">
      {services.map((svc, index) => (
        <ServiceCard key={svc.id} service={svc} index={index} />
      ))}
    </ul>
  </div>
);
