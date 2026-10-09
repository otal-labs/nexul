import { Text } from "@/components/ui/text";

// The solid ink pill for an unread count: a muted pill vanished on the light canvas and a bare number read as a count of items.
export const UnreadBadge = ({ count }: { count: number }) => (
  <Text
    aria-label={`${count} unread`}
    className="min-w-5 overflow-hidden rounded-full bg-foreground px-1.5 text-center font-mono text-[11px] leading-5 text-background"
  >
    {count > 99 ? "99+" : count}
  </Text>
);
