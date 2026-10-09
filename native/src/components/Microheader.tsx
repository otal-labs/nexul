import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";

interface MicroheaderProps {
  children: string;
  className?: string;
}

// The one small uppercase label: group heads, facts, section heads (the web's microheaderClass).
export const Microheader = ({ children, className }: MicroheaderProps) => (
  <Text className={cn("font-mono text-[11px] font-medium uppercase tracking-[1.3px] text-muted-foreground", className)}>
    {children}
  </Text>
);
