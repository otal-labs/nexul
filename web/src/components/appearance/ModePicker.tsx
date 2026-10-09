import { RadioGroup as RadioGroupPrimitive } from "radix-ui";
import { useShallow } from "zustand/react/shallow";

import { ThemeMiniature } from "@/components/appearance/ThemeMiniature";
import { ThemeTile } from "@/components/appearance/ThemeTile";
import { AppearanceMode, type AppearanceMode as AppearanceModeType } from "@/enums/Theme";
import { themePreviewColors } from "@/lib/themePalettes";
import { useThemeStore } from "@/stores/themeStore";

const MODES = [
  { mode: AppearanceMode.System, label: "System" },
  { mode: AppearanceMode.Light, label: "Light" },
  { mode: AppearanceMode.Dark, label: "Dark" },
] as const;

// Each mode drawn in the current palette; System is light on the left and dark on the right, as the OS decides.
export const ModePicker = () => {
  const { appearanceMode, themeId, setAppearanceMode } = useThemeStore(
    useShallow((s) => ({ appearanceMode: s.appearanceMode, themeId: s.themeId, setAppearanceMode: s.setAppearanceMode })),
  );
  const light = themePreviewColors(themeId, "light");
  const dark = themePreviewColors(themeId, "dark");

  const pick = (mode: string) => setAppearanceMode(mode as AppearanceModeType);

  return (
    <RadioGroupPrimitive.Root aria-label="Mode" value={appearanceMode} onValueChange={pick} className="grid grid-cols-3 gap-3">
      {MODES.map(({ mode, label }) => (
        <ThemeTile key={mode} value={mode} label={label} selected={appearanceMode === mode}>
          {mode === AppearanceMode.Light && <ThemeMiniature colors={light} />}
          {mode === AppearanceMode.Dark && <ThemeMiniature colors={dark} />}
          {mode === AppearanceMode.System && (
            <span className="grid">
              <span className="col-start-1 row-start-1">
                <ThemeMiniature colors={light} />
              </span>
              <span className="col-start-1 row-start-1 [clip-path:polygon(55%_0,100%_0,100%_100%,45%_100%)]">
                <ThemeMiniature colors={dark} />
              </span>
            </span>
          )}
        </ThemeTile>
      ))}
    </RadioGroupPrimitive.Root>
  );
};
