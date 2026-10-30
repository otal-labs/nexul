import { lazy, Suspense, type CSSProperties } from "react";

import { prefersReducedMotion } from "@/lib/motion";
import { cn } from "@/lib/utils";

const LiveLightField = lazy(() => import("@/components/showcase/LiveLightField").then((m) => ({ default: m.LiveLightField })));

interface ShowcaseFieldProps {
  live?: boolean;
  quiet?: boolean;
  className?: string;
}

const hasWebGPU = typeof navigator !== "undefined" && "gpu" in navigator;

// Mid-grey noise overlaid on the glows: a quiet glow spans about three levels per channel, so without it each step draws a ring.
const STILL_GRAIN =
  "url(\"data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='160' height='160'%3E%3Cfilter id='g' color-interpolation-filters='sRGB'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.8' numOctaves='2' stitchTiles='stitch'/%3E%3CfeColorMatrix values='0.4 0 0 0 0.3 0.4 0 0 0 0.3 0.4 0 0 0 0.3 0 0 0 0 1'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23g)'/%3E%3C/svg%3E\")";

// The still is the light field's own gradients at the live layer's strength, placed where its colours sit.
const still = (strength: number): CSSProperties => ({
  backgroundColor: "var(--background)",
  backgroundBlendMode: "overlay, normal, normal, normal",
  backgroundImage: [
    STILL_GRAIN,
    `radial-gradient(52% 56% at 94% 4%, oklch(from var(--field-warm) l c h / ${strength}), transparent 72%)`,
    `radial-gradient(36% 40% at 84% 40%, oklch(from var(--field-mid) l c h / ${strength}), transparent 70%)`,
    `radial-gradient(56% 60% at 4% 96%, oklch(from var(--field-low) l c h / ${strength}), transparent 72%)`,
  ].join(","),
});

// The live layer repaints every frame, so callers opt in; the still is its fallback and reduced-motion variant.
export const ShowcaseField = ({ live = false, quiet = false, className }: ShowcaseFieldProps) => (
  <div aria-hidden style={still(quiet ? 0.22 : 0.46)} className={cn("pointer-events-none absolute inset-0 overflow-hidden", className)}>
    {live && hasWebGPU && !prefersReducedMotion() && (
      <Suspense>
        <LiveLightField quiet={quiet} />
      </Suspense>
    )}
  </div>
);
