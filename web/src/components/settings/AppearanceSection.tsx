import { useShallow } from "zustand/react/shallow";

import { ModePicker } from "@/components/appearance/ModePicker";
import { ThemeMiniature } from "@/components/appearance/ThemeMiniature";
import { ThemePicker } from "@/components/appearance/ThemePicker";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { SettingsRow, SettingsRows } from "@/components/settings/SettingsRow";
import { Slider } from "@/components/ui/slider";
import { themePreviewColors } from "@/lib/themePalettes";
import { MAX_GLASS_OPACITY, MIN_GLASS_OPACITY, useThemeStore } from "@/stores/themeStore";

// Every choice here applies the moment it is made and stays in this browser; there is nothing to save.
export const AppearanceSection = () => {
  const { themeId, theme, glassOpacity, setGlassOpacity } = useThemeStore(
    useShallow((s) => ({ themeId: s.themeId, theme: s.theme, glassOpacity: s.glassOpacity, setGlassOpacity: s.setGlassOpacity })),
  );

  return (
    <>
      <SettingsCard id="appearance-mode" title="Mode" description="Applies at once, in this browser.">
        <ModePicker />
      </SettingsCard>
      <SettingsCard id="appearance-theme" title="Theme" description="Recolours the accent, the surfaces and the light behind them.">
        <ThemePicker />
      </SettingsCard>
      <SettingsCard id="appearance-backdrop" title="Dialog backdrop">
        <SettingsRows>
          <SettingsRow label="Dim behind dialogs" description="How much a dialog darkens the page under it." htmlFor="glass-opacity">
            <div className="flex w-full items-center gap-4">
              <span className="w-24 shrink-0 overflow-hidden rounded-md ring-1 ring-border">
                <ThemeMiniature colors={themePreviewColors(themeId, theme)} scrim={glassOpacity / 100} />
              </span>
              <Slider
                id="glass-opacity"
                aria-label="Dim behind dialogs"
                min={MIN_GLASS_OPACITY}
                max={MAX_GLASS_OPACITY}
                step={5}
                value={[glassOpacity]}
                onValueChange={([value]) => value !== undefined && setGlassOpacity(value)}
                className="flex-1"
              />
              <output htmlFor="glass-opacity" className="w-10 shrink-0 text-right font-mono text-xs text-muted-foreground tabular-nums">
                {glassOpacity}%
              </output>
            </div>
          </SettingsRow>
        </SettingsRows>
      </SettingsCard>
    </>
  );
};
