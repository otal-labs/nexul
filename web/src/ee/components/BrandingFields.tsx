import { Controller, type Control } from "react-hook-form";

import { FormInput } from "@/components/FormInput";
import { ColorPicker } from "@/components/settings/ColorPicker";
import { SettingsRow } from "@/components/settings/SettingsRow";
import { Button } from "@/components/ui/button";
import { useLook } from "@/ee/hooks/BrandingHooks";
import { DEFAULT_PRODUCT_NAME, type BrandingFormData } from "@/ee/models/Branding";

type FieldProps = { control: Control<BrandingFormData>; disabled: boolean };

// The picked accent next to what light mode paints from it, since a bright color is darkened there to stay readable.
const LightResult = () => {
  const { data: look } = useLook();
  const brand = look?.enabled ? look.light.roles.brand : undefined;
  return (
    brand && (
      <span className="flex items-center gap-1.5 text-xs text-muted-foreground" title="How it paints in light mode">
        <span aria-hidden>→</span>
        <span className="size-4 rounded-full border border-input" style={{ backgroundColor: brand }} />
        <span className="sr-only">In light mode</span>
      </span>
    )
  );
};

export const BrandingNameRow = ({ control, disabled }: FieldProps) => (
  <SettingsRow label="Product name" description={`Shown instead of ${DEFAULT_PRODUCT_NAME} in the sidebar, on sign-in, in the browser tab and in what the instance sends.`} htmlFor="branding-name">
    <div className="w-full">
      <FormInput control={control} name="product_name" id="branding-name" label="Product name" hideLabel placeholder={DEFAULT_PRODUCT_NAME} maxLength={40} disabled={disabled} />
    </div>
  </SettingsRow>
);

// The accent is the one `brand` colour; the server moves it if it would not read, and says so in the theme notes.
export const BrandingAccentRow = ({ control, disabled, fromTheme }: FieldProps & { fromTheme: boolean }) => (
  <SettingsRow label="Accent" description="The one color for actions, the current page and selection. Darkened in light mode to stay readable.">
    <Controller
      control={control}
      name="accent"
      render={({ field }) => (
        <div className="flex w-full flex-wrap items-center gap-x-2 gap-y-2">
          <ColorPicker label="Accent" value={field.value} onChange={field.onChange} disabled={disabled} emptyLabel={fromTheme ? "From theme" : "Default"} />
          {field.value !== "" && <LightResult />}
          {field.value !== "" && (
            <Button type="button" variant="ghost" size="sm" className="disabled:opacity-50" disabled={disabled} onClick={() => field.onChange("")}>
              Use the default
            </Button>
          )}
        </div>
      )}
    />
  </SettingsRow>
);
