import type { ReactNode } from "react";

import { TrailStepDetail } from "@/components/play/TrailStepDetail";

interface StepDetailDisclosureProps {
  detail: string;
  at: string;
  children: ReactNode;
}

// A step's line that opens to when it happened, its arguments and its result.
export const StepDetailDisclosure = ({ detail, at, children }: StepDetailDisclosureProps) => (
  <details>
    <summary className="cursor-pointer list-none rounded-md [&::-webkit-details-marker]:hidden">{children}</summary>
    <TrailStepDetail detail={detail} at={at} />
  </details>
);
