import { toast } from "sonner";

import { ActionButton } from "@/components/action/ActionButton";
import { SettingsRow } from "@/components/settings/SettingsRow";
import { BrandingFileButton } from "@/ee/components/BrandingFileButton";
import { useUpdateBranding } from "@/ee/hooks/BrandingHooks";
import { Appearance, themeFor, type Branding, type BrandingTheme } from "@/ee/models/Branding";

const MAX_THEME_BYTES = 1024 * 1024;

const Swatches = ({ theme }: { theme: BrandingTheme }) => (
  <span className="flex shrink-0 items-center" aria-hidden>
    {[theme.canvas, theme.text, theme.accent].map((color, i) => (
      <span key={i} className="-ml-1 size-4 rounded-full ring-2 ring-card first:ml-0" style={{ backgroundColor: color }} />
    ))}
  </span>
);

interface ThemeRowProps {
  mode: Appearance;
  branding: Branding;
}

const readableNote = (adjusted: string[]): string => `Moved to stay readable: ${adjusted.join(", ")}.`;

// One imported theme: the mode it paints, what it holds, and the way back to the default.
const ThemeRow = ({ mode, branding }: ThemeRowProps) => {
  const update = useUpdateBranding();
  const theme = themeFor(branding, mode);
  const locked = !branding.unlocked;
  const label = mode === Appearance.Dark ? "Dark theme" : "Light theme";
  const adjusted = theme?.adjusted ?? [];

  const choose = async (file: File) => {
    if (file.size > MAX_THEME_BYTES) return void toast.error("A theme file is at most 1 MB");
    update.mutate({ theme_file: await file.text() });
  };

  return (
    <SettingsRow
      label={label}
      description={
        <>
          {theme ? `${theme.name}.` : "The standard colors."} A VS Code color theme file (.json), saved when you choose it.
          {adjusted.length > 0 && <span className="mt-1 block">{readableNote(adjusted)}</span>}
        </>
      }
    >
      <div className="flex w-full min-w-0 items-center gap-2">
        {theme && <Swatches theme={theme} />}
        <div className="flex flex-wrap gap-x-1 gap-y-4">
          <BrandingFileButton label={theme ? "Replace" : "Import"} accept=".json,.jsonc,application/json" verb="import a theme" disabled={locked} busy={update.isPending} onFile={(f) => void choose(f)} />
          {theme && (
            <ActionButton type="button" permission="instance:write" verb="remove the theme" size="sm" variant="ghost" onClick={() => update.mutate({ remove_theme: mode })}>
              Remove
            </ActionButton>
          )}
        </div>
      </div>
    </SettingsRow>
  );
};

export const BrandingThemeRows = ({ branding }: { branding: Branding }) => (
  <>
    <ThemeRow mode={Appearance.Dark} branding={branding} />
    <ThemeRow mode={Appearance.Light} branding={branding} />
  </>
);
