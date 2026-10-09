import type { HTMLAttributes } from "react";

import { cn } from "@/lib/utils";

interface ContainerProps extends HTMLAttributes<HTMLDivElement> {
  // "wide" for lists and grids, "page" for settings and single-record pages.
  size?: "wide" | "page";
}

export const Container = ({ className, size = "wide", ...props }: ContainerProps) => (
  <div
    className={cn("mx-auto w-full px-4 sm:px-6", size === "wide" ? "max-w-7xl" : "max-w-5xl", className)}
    {...props}
  />
);
