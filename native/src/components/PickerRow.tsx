import { Check } from "lucide-react-native";
import type { ReactNode } from "react";
import { Pressable } from "react-native";
import { useCSSVariable } from "uniwind";

import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";

interface PickerRowProps {
  label: string;
  selected: boolean;
  onPress: () => void;
  leading?: ReactNode;
  meta?: string;
  disabled?: boolean;
}

// One choice in a sheet; the current one is the selection, so its check is the ember.
export const PickerRow = ({ label, selected, onPress, leading, meta, disabled = false }: PickerRowProps) => {
  const [brand] = useCSSVariable(["--color-brand"]);
  return (
    <Pressable
      role="radio"
      aria-checked={selected}
      disabled={disabled}
      onPress={onPress}
      className={cn("min-h-[52px] flex-row items-center gap-3 px-5 py-2.5 active:bg-accent", disabled && "opacity-50")}
    >
      {leading}
      <Text numberOfLines={1} className={cn("min-w-0 flex-1 text-[15px]", selected && "font-medium")}>
        {label}
      </Text>
      {meta && <Text className="font-mono text-xs text-muted-foreground">{meta}</Text>}
      <Check size={18} color={String(brand)} style={{ opacity: selected ? 1 : 0 }} />
    </Pressable>
  );
};
