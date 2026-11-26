import { useRouter } from "expo-router";

import { SheetBody } from "@/components/SheetBody";
import { PickerRow } from "@/components/PickerRow";
import { SheetTitle } from "@/components/SheetTitle";
import { appearanceLabel, useAppearanceStore, type Appearance } from "@/stores/appearanceStore";

const OPTIONS: Appearance[] = ["system", "light", "dark"];

export const AppearanceScreen = () => {
  const router = useRouter();
  const appearance = useAppearanceStore((s) => s.appearance);
  const setAppearance = useAppearanceStore((s) => s.setAppearance);

  const choose = (value: Appearance) => {
    setAppearance(value);
    router.back();
  };

  return (
    <SheetBody role="radiogroup">
      <SheetTitle title="Appearance" />
      {OPTIONS.map((value) => (
        <PickerRow key={value} label={appearanceLabel[value]} selected={appearance === value} onPress={() => choose(value)} />
      ))}
    </SheetBody>
  );
};
