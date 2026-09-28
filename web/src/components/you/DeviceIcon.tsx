import { Monitor, Smartphone } from "lucide-react";

import type { SessionClient } from "@/models/User";

export const DeviceIcon = ({ client }: { client: SessionClient }) => (
  <span className="flex size-9 shrink-0 items-center justify-center rounded-md border border-border bg-muted/60 text-muted-foreground">
    {client === "phone" && <Smartphone className="size-4" aria-hidden />}
    {client !== "phone" && <Monitor className="size-4" aria-hidden />}
  </span>
);
