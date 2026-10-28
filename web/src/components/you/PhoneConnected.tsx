import { CircleCheck } from "lucide-react";

import { SettingsHint } from "@/components/settings/SettingsHint";

interface PhoneConnectedProps {
  label: string;
}

// The one hero moment: an 800ms crossfade well past the 150 to 250ms baseline, because it happens once per phone.
export const PhoneConnected = ({ label }: PhoneConnectedProps) => (
  <div className="@container w-full">
    <div className="flex min-h-[150px] animate-in flex-col items-start gap-5 fade-in-0 blur-in-[4px] zoom-in-[0.98] duration-800 ease-out @sm:flex-row @sm:items-center">
      <span className="flex size-[150px] shrink-0 items-center justify-center rounded-md border border-border bg-muted/40">
        <CircleCheck className="size-10 text-success" aria-hidden />
      </span>
      <div className="min-w-0 space-y-3 text-sm">
        <p className="font-medium">{label} is connected</p>
        <SettingsHint>It's in your devices below. Its code no longer works.</SettingsHint>
      </div>
    </div>
  </div>
);
