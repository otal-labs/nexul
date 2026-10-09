import type { ReactNode } from "react";
import { View } from "react-native";

import { Microheader } from "@/components/Microheader";
import { cn } from "@/lib/utils";

interface SettingsCardProps {
  title?: string;
  children: ReactNode;
  className?: string;
}

// A group of rows on one card, an optional microheader above it.
export const SettingsCard = ({ title, children, className }: SettingsCardProps) => (
  <View className={cn("gap-2", className)}>
    {title && <Microheader className="px-1">{title}</Microheader>}
    <View className="overflow-hidden rounded-xl border border-border bg-card">{children}</View>
  </View>
);
