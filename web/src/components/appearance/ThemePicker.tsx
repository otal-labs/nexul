import { RadioGroup as RadioGroupPrimitive } from "radix-ui";
import { useShallow } from "zustand/react/shallow";

import { ThemeMiniature } from "@/components/appearance/ThemeMiniature";
import { ThemeTile } from "@/components/appearance/ThemeTile";
import type { ThemeId } from "@/enums/Theme";
import { THEME_DEFINITIONS, themePreviewColors } from "@/lib/themePalettes";
import { useThemeStore } from "@/stores/themeStore";

// Every palette drawn as the app in miniature, in the mode the app is showing now.
export const ThemePicker = () => {
  const { themeId, theme, setThemeId } = useThemeStore(
    useShallow((s) => ({ themeId: s.themeId, theme: s.theme, setThemeId: s.setThemeId })),
  );

  const pick = (id: string) => setThemeId(id as ThemeId);

  return (
    <RadioGroupPrimitive.Root aria-label="Theme" value={themeId} onValueChange={pick} className="grid grid-cols-3 gap-3 @lg:grid-cols-4">
      {THEME_DEFINITIONS.map((definition) => (
        <ThemeTile key={definition.id} value={definition.id} label={definition.label} selected={themeId === definition.id}>
          <ThemeMiniature colors={themePreviewColors(definition.id, theme)} />
        </ThemeTile>
      ))}
    </RadioGroupPrimitive.Root>
  );
};
