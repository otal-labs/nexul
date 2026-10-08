import { Monitor, Moon, Sun } from "lucide-react";
import { useShallow } from "zustand/react/shallow";

import { SettingsCard } from "@/components/settings/SettingsCard";
import { Slider } from "@/components/ui/slider";
import { AppearanceMode } from "@/enums/Theme";
import { THEME_DEFINITIONS } from "@/lib/themePalettes";
import { cn } from "@/lib/utils";
import { MAX_GLASS_OPACITY, MIN_GLASS_OPACITY, useThemeStore } from "@/stores/themeStore";

const MODE_OPTIONS = [
  { mode: AppearanceMode.System, label: "System", icon: Monitor },
  { mode: AppearanceMode.Light, label: "Light", icon: Sun },
  { mode: AppearanceMode.Dark, label: "Dark", icon: Moon },
] as const;

// CSS-variable-driven theme settings; see themePalettes.ts for how a theme applies.
export const AppearanceSection = () => {
  const { appearanceMode, themeId, glassOpacity, setAppearanceMode, setThemeId, setGlassOpacity } = useThemeStore(
    useShallow((s) => ({
      appearanceMode: s.appearanceMode,
      themeId: s.themeId,
      glassOpacity: s.glassOpacity,
      setAppearanceMode: s.setAppearanceMode,
      setThemeId: s.setThemeId,
      setGlassOpacity: s.setGlassOpacity,
    })),
  );

  return (
    <SettingsCard id="appearance" title="Appearance" description="Choose how Nexul looks.">
      <div className="space-y-6">
        <div>
          <h3 className="text-sm font-semibold">Color scheme</h3>
          <div role="radiogroup" aria-label="Color scheme" className="mt-3 grid grid-cols-3 gap-2">
            {MODE_OPTIONS.map(({ mode, label, icon: Icon }) => {
              const active = appearanceMode === mode;
              return (
                <button
                  key={mode}
                  type="button"
                  role="radio"
                  aria-checked={active}
                  onClick={() => setAppearanceMode(mode)}
                  className={cn(
                    "flex flex-col items-center gap-1.5 rounded-md border p-3 text-sm transition-colors duration-150 ease-standard",
                    active
                      ? "border-primary bg-accent/60 text-foreground"
                      : "border-border text-muted-foreground hover:bg-accent/30",
                  )}
                >
                  <Icon className="size-4" aria-hidden />
                  {label}
                </button>
              );
            })}
          </div>
        </div>

        <div className="border-t pt-6">
          <h3 className="text-sm font-semibold">Themes</h3>
          <div role="radiogroup" aria-label="Theme" className="mt-3 grid grid-cols-2 gap-2 sm:grid-cols-4">
            {THEME_DEFINITIONS.map((theme) => {
              const active = themeId === theme.id;
              return (
                <button
                  key={theme.id}
                  type="button"
                  role="radio"
                  aria-checked={active}
                  onClick={() => setThemeId(theme.id)}
                  className={cn(
                    "flex flex-col items-center gap-2 rounded-md border p-3 transition-colors duration-150 ease-standard",
                    active ? "border-primary bg-accent/60" : "border-border hover:bg-accent/30",
                  )}
                >
                  <span className="flex -space-x-1.5">
                    <span
                      className="size-5 rounded-full border border-border"
                      style={{ background: theme.swatch[0] }}
                      aria-hidden
                    />
                    <span
                      className="size-5 rounded-full border border-border"
                      style={{ background: theme.swatch[1] }}
                      aria-hidden
                    />
                  </span>
                  <span className="text-xs font-medium text-foreground">{theme.label}</span>
                </button>
              );
            })}
          </div>
        </div>

        <div className="border-t pt-6">
          <h3 className="text-sm font-semibold">Glass opacity</h3>
          <p className="mt-1 text-sm text-muted-foreground">
            Control how transparent glass surfaces are. Higher values make dialogs and menus more solid.
          </p>
          <div className="mt-3 flex items-center gap-3 sm:w-64">
            <Slider
              aria-label="Glass opacity"
              min={MIN_GLASS_OPACITY}
              max={MAX_GLASS_OPACITY}
              step={5}
              value={[glassOpacity]}
              onValueChange={([value]) => value !== undefined && setGlassOpacity(value)}
              className="flex-1"
            />
            <output className="w-10 shrink-0 text-right text-sm text-muted-foreground">{glassOpacity}%</output>
          </div>
        </div>
      </div>
    </SettingsCard>
  );
};
