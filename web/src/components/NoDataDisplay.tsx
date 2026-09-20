import { EmptyState } from "@/components/EmptyState";

interface NoDataDisplayProps {
  message?: string;
  className?: string;
  size?: "default" | "compact";
}

export const NoDataDisplay = ({
  message = "Nothing here yet",
  className,
  size,
}: NoDataDisplayProps) => (
  <EmptyState
    role="status"
    title={message}
    {...(className ? { className } : {})}
    {...(size ? { size } : {})}
  />
);
