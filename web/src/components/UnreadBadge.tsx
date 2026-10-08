import { cn } from "@/lib/utils";

interface UnreadBadgeProps {
  count: number;
  className?: string | undefined;
}

export const UnreadBadge = ({ count, className }: UnreadBadgeProps) =>
  count > 0 && (
    <span
      className={cn(
        "flex h-4 min-w-4 shrink-0 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-semibold text-primary-foreground tabular-nums",
        className,
      )}
    >
      {count > 99 ? "99+" : count}
    </span>
  );
