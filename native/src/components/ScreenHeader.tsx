import type { ReactNode } from "react";
import { View } from "react-native";

import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";

interface ScreenHeaderProps {
  title: string;
  // The workspace or project it belongs to, as a small mono line above the title.
  eyebrow?: string | undefined;
  // One muted line under the title: counts, status, who and when.
  meta?: ReactNode;
  // A mark that names the record (the board's project mark), centred on the title and meta.
  leading?: ReactNode;
  // The screen's one action, on the title's line.
  action?: ReactNode;
  className?: string;
}

// A long title steps down so a sentence-long ticket still fits a phone in three lines.
const titleSize = (title: string) => {
  if (title.length > 90) return "text-[22px] leading-[27px]";
  if (title.length > 44) return "text-[25px] leading-[30px]";
  return "text-[30px] leading-[35px]";
};

// The top of a screen in the web's page-header grammar: eyebrow, the title in the display face, one meta line.
export const ScreenHeader = ({ title, eyebrow, meta, leading, action, className }: ScreenHeaderProps) => (
  <View className={cn("gap-1.5 px-5 pb-4 pt-5", className)}>
    {eyebrow && (
      <Text numberOfLines={1} className="font-mono text-xs text-muted-foreground">
        {eyebrow}
      </Text>
    )}
    <View className="flex-row items-center gap-3">
      {leading}
      <View className="min-w-0 flex-1 gap-1.5">
        <Text role="heading" numberOfLines={3} className={cn("font-display tracking-[-0.5px] text-foreground", titleSize(title))}>
          {title}
        </Text>
        {typeof meta === "string" && <Text className="text-sm text-muted-foreground">{meta}</Text>}
        {!!meta && typeof meta !== "string" && <View className="flex-row flex-wrap items-center gap-x-2">{meta}</View>}
      </View>
      {action && <View className="self-start">{action}</View>}
    </View>
  </View>
);
