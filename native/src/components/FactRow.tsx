import type { ReactNode } from "react";
import { View } from "react-native";

import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";

interface FactRowProps {
  label: string;
  children: ReactNode;
  first?: boolean;
}

// One property inside a card: a muted label on the left, the value on the right, a hairline between rows.
export const FactRow = ({ label, children, first = false }: FactRowProps) => (
  <View className={cn("min-h-12 flex-row items-center gap-3 px-4 py-2.5", !first && "border-t border-border")}>
    <Text className="min-w-24 shrink-0 text-[13px] text-muted-foreground">{label}</Text>
    <View className="min-w-0 flex-1 flex-row items-center justify-end gap-2">{children}</View>
  </View>
);
