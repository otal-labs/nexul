import { useRouter } from "expo-router";
import { Check } from "lucide-react-native";
import { Pressable, View } from "react-native";
import { useCSSVariable } from "uniwind";

import { Text } from "@/components/ui/text";
import { appearanceLabel, useAppearanceStore, type Appearance } from "@/stores/appearanceStore";

const OPTIONS: Appearance[] = ["system", "light", "dark"];

export const AppearanceScreen = () => {
  const router = useRouter();
  const appearance = useAppearanceStore((s) => s.appearance);
  const setAppearance = useAppearanceStore((s) => s.setAppearance);
  const [foreground] = useCSSVariable(["--color-foreground"]);

  const choose = (value: Appearance) => {
    setAppearance(value);
    router.back();
  };

  return (
    <View className="flex-1 bg-background pt-2">
      {OPTIONS.map((value) => (
        <Pressable
          key={value}
          role="button"
          onPress={() => choose(value)}
          className="min-h-11 flex-row items-center justify-between gap-2 border-b border-border bg-card px-4 py-3 active:bg-accent"
        >
          <Text className="font-medium">{appearanceLabel[value]}</Text>
          {appearance === value && <Check size={16} color={String(foreground)} />}
        </Pressable>
      ))}
    </View>
  );
};
