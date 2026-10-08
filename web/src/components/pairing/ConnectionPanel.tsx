import { Laptop } from "lucide-react";
import { useReducedMotion } from "motion/react";
import type { ReactNode } from "react";

import { Logo } from "@/components/Logo";
import { cn } from "@/lib/utils";

const VB = { w: 260, h: 90 };
const LX = 44;
const RX = 216;
const Y = VB.h / 2;
const FORWARD = `M ${LX},${Y} C ${LX + 60},${Y - 18} ${RX - 60},${Y - 18} ${RX},${Y}`;
const REVERSE = `M ${RX},${Y} C ${RX - 60},${Y + 18} ${LX + 60},${Y + 18} ${LX},${Y}`;

interface PathPulseProps {
  d: string;
  begin: string;
}

// SMIL ignores the reduced-motion CSS block, so the caller only mounts pulses when motion is allowed.
const PathPulse = ({ d, begin }: PathPulseProps) => (
  <circle r={2.5} fill="currentColor" className="text-foreground">
    <animateMotion dur="2.4s" repeatCount="indefinite" begin={begin} path={d} />
  </circle>
);

interface EndpointProps {
  x: number;
  label: string;
  children: ReactNode;
}

const Endpoint = ({ x, label, children }: EndpointProps) => (
  <div
    className="absolute flex -translate-x-1/2 -translate-y-1/2 flex-col items-center gap-1.5"
    style={{ left: `${(x / VB.w) * 100}%`, top: `${(Y / VB.h) * 100}%` }}
  >
    <span className="grid size-11 place-items-center rounded-lg border border-border bg-card shadow-card">{children}</span>
    <span className="absolute top-full mt-1.5 text-[11px] whitespace-nowrap text-muted-foreground">{label}</span>
  </div>
);

interface ConnectionPanelProps {
  connected: boolean;
  hostname: string;
}

// This computer and Nexul, joined by the tunnel: idle and dashed while waiting, traffic flowing once both checks pass.
export const ConnectionPanel = ({ connected, hostname }: ConnectionPanelProps) => {
  const reduceMotion = useReducedMotion();
  return (
    <div className="flex flex-col items-center gap-4 rounded-lg border border-border bg-surface-2 px-4 pt-4 pb-3">
      <div className="relative aspect-[26/9] w-full max-w-64">
        <svg className="pointer-events-none absolute inset-0 size-full" viewBox={`0 0 ${VB.w} ${VB.h}`} fill="none" aria-hidden>
          <path
            d={FORWARD}
            stroke="currentColor"
            strokeWidth={1}
            strokeDasharray={connected ? undefined : "5 4"}
            strokeLinecap="round"
            className={cn("transition-colors duration-150 ease-standard", connected ? "text-foreground/40" : "text-muted-foreground/50")}
          />
          {connected && !reduceMotion && (
            <g>
              <PathPulse d={FORWARD} begin="0s" />
              <PathPulse d={REVERSE} begin="-1.2s" />
            </g>
          )}
        </svg>
        <Endpoint x={LX} label="This computer">
          <Laptop className="size-5 text-foreground" aria-hidden />
        </Endpoint>
        <Endpoint x={RX} label="Nexul">
          <Logo className="size-7" />
        </Endpoint>
      </div>
      <div className="mt-3 flex flex-col items-center gap-1 text-center" role="status">
        <span className="flex items-center gap-1.5 text-sm">
          <span
            className={cn(
              "size-1.5 rounded-full transition-colors duration-150 ease-standard",
              connected ? "bg-success" : "bg-muted-foreground/60 animate-[status-pulse_2.4s_ease-standard_infinite]",
            )}
            aria-hidden
          />
          {connected ? "Connected" : "Waiting for connection…"}
        </span>
        <span className="max-w-full font-mono text-[11px] break-all text-muted-foreground">{hostname}</span>
      </div>
    </div>
  );
};
