import { CircleCheck } from "lucide-react";

import type { ProtoDevice } from "@/components/you/prototypeStore";

interface PhoneConnectedProps {
  phone: ProtoDevice;
}

export const PhoneConnected = ({ phone }: PhoneConnectedProps) => (
  <div className="flex min-h-[150px] animate-in items-center gap-5 fade-in-0 blur-in-[4px] zoom-in-[0.98] duration-800 ease-out">
    <span className="flex size-[150px] shrink-0 items-center justify-center rounded-md border border-border bg-muted/40">
      <CircleCheck className="size-10 text-success" aria-hidden />
    </span>
    <div className="min-w-0 space-y-3 text-sm">
      <p className="font-medium">{phone.label.split(" · ")[1]} is connected</p>
      <p className="text-muted-foreground">It's in your device list below. The code it used no longer works.</p>
    </div>
  </div>
);
