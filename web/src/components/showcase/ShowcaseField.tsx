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

// The still is the light field's own gradients at the live layer's strength, placed where its colours sit.
const still = (strength: number): CSSProperties => ({
  backgroundColor: "var(--background)",
  backgroundImage: [
    `radial-gradient(52% 56% at 94% 4%, oklch(from var(--brand) l c h / ${strength}), transparent 72%)`,
    `radial-gradient(36% 40% at 84% 40%, oklch(from var(--field-pink) l c h / ${strength}), transparent 70%)`,
    `radial-gradient(56% 60% at 4% 96%, oklch(from var(--field-cool) l c h / ${strength}), transparent 72%)`,
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
