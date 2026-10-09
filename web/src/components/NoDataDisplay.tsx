import type { ComponentType } from "react";

import { EmptyState } from "@/components/EmptyState";

interface NoDataDisplayProps {
  icon?: ComponentType<{ className?: string }>;
  message?: string;
  className?: string;
  size?: "default" | "compact";
}

export const NoDataDisplay = ({
  icon,
  message = "Nothing here yet",
  className,
  size,
}: NoDataDisplayProps) => (
  <EmptyState
    role="status"
    title={message}
    {...(icon ? { icon } : {})}
    {...(className ? { className } : {})}
    {...(size ? { size } : {})}
  />
);
