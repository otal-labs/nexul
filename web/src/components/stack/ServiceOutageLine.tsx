import { XCircle } from "lucide-react";

import type { ServiceOutage } from "@/models/Stack";
import { outageText } from "@/utils/OutageUtility";

interface ServiceOutageLineProps {
  outage: ServiceOutage;
}

// Status is an icon in its own hue beside plain text.
export const ServiceOutageLine = ({ outage }: ServiceOutageLineProps) => (
  <p className="flex items-center gap-1.5 text-xs text-foreground">
    <XCircle className="size-3.5 shrink-0 text-destructive" aria-hidden />
    <span className="min-w-0 wrap-anywhere">{outageText(outage)}</span>
  </p>
);
