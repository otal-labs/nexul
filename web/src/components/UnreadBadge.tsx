import { cn } from "@/lib/utils";

interface UnreadBadgeProps {
  count: number;
  className?: string | undefined;
}

export const UnreadBadge = ({ count, className }: UnreadBadgeProps) =>
  count > 0 && (
    <span
      className={cn(
        "shrink-0 font-mono text-xs leading-none text-muted-foreground tabular-nums",
        className,
      )}
    >
      {count > 99 ? "99+" : count}
    </span>
  );
