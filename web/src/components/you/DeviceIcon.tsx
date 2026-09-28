import { Monitor, Smartphone } from "lucide-react";

import type { Session } from "@/models/User";

const PHONE_PLATFORMS = ["Android", "iOS"];

export const DeviceIcon = ({ session }: { session: Pick<Session, "client" | "platform"> }) => {
  const phone = session.client === "phone" || PHONE_PLATFORMS.includes(session.platform);
  return (
    <span className="flex size-9 shrink-0 items-center justify-center rounded-md border border-border bg-muted/60 text-muted-foreground">
      {phone && <Smartphone className="size-4" aria-hidden />}
      {!phone && <Monitor className="size-4" aria-hidden />}
    </span>
  );
};
