import type { HTMLAttributes } from "react";

import { cn } from "@/lib/utils";

interface ContainerProps extends HTMLAttributes<HTMLDivElement> {
  // "wide" for lists, grids and the records they open, so the content edge holds from a list to its record; "page" for settings.
  size?: "wide" | "page";
}

export const Container = ({ className, size = "wide", ...props }: ContainerProps) => (
  <div
    className={cn("mx-auto w-full px-4 sm:px-6", size === "wide" ? "max-w-7xl" : "max-w-5xl", className)}
    {...props}
  />
);
