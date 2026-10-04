import { ChevronLeft, ChevronRight } from "lucide-react-native";
import { Pressable, View } from "react-native";

import { Icon } from "@/components/ui/icon";
import { Text } from "@/components/ui/text";

interface CyclerProps {
  label: string;
  onStep: (by: number) => void;
}

const Cycler = ({ label, onStep }: CyclerProps) => (
  <View className="min-w-0 flex-1 flex-row items-center">
    <Pressable role="button" aria-label="Previous" onPress={() => onStep(-1)} className="h-12 w-11 items-center justify-center active:bg-accent">
      <Icon as={ChevronLeft} size={18} />
    </Pressable>
    <Text numberOfLines={2} className="min-w-0 flex-1 text-center font-mono text-xs leading-snug">
      {label}
    </Text>
    <Pressable role="button" aria-label="Next" onPress={() => onStep(1)} className="h-12 w-11 items-center justify-center active:bg-accent">
      <Icon as={ChevronRight} size={18} />
    </Pressable>
  </View>
);

interface ClarifyProtoSwitcherProps {
  variant: string;
  state: string;
  onVariant: (by: number) => void;
  onState: (by: number) => void;
}

// Dev-only: a dashed strip above the screen that cycles the variant and the state; not part of any design.
export const ClarifyProtoSwitcher = ({ variant, state, onVariant, onState }: ClarifyProtoSwitcherProps) => (
  <View className="-mx-4 -mt-4 mb-4 border-b border-dashed border-muted-foreground bg-surface-2">
    <Text className="pt-1.5 text-center font-mono text-[10px] uppercase leading-snug text-muted-foreground">Prototype</Text>
    <View className="flex-row">
      <Cycler label={variant} onStep={onVariant} />
      <Cycler label={state} onStep={onState} />
    </View>
  </View>
);
