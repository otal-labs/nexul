import { Controller, type Control } from "react-hook-form";

import { FormInput } from "@/components/FormInput";
import { ColorPicker } from "@/components/settings/ColorPicker";
import { SettingsRow } from "@/components/settings/SettingsRow";
import { Button } from "@/components/ui/button";
import { DEFAULT_PRODUCT_NAME, type BrandingFormData } from "@/ee/models/Branding";

type FieldProps = { control: Control<BrandingFormData>; disabled: boolean };

export const BrandingNameRow = ({ control, disabled }: FieldProps) => (
  <SettingsRow label="Product name" description={`Shown instead of ${DEFAULT_PRODUCT_NAME} in the sidebar, on sign-in, in the browser tab and in what the instance sends.`} htmlFor="branding-name">
    <div className="w-full">
      <FormInput control={control} name="product_name" id="branding-name" label="Product name" hideLabel placeholder={DEFAULT_PRODUCT_NAME} maxLength={40} disabled={disabled} />
    </div>
  </SettingsRow>
);

// The accent is the one `brand` colour; the server moves it if it would not read, and says so in the theme notes.
export const BrandingAccentRow = ({ control, disabled }: FieldProps) => (
  <SettingsRow label="Accent" description="The one color for actions, the current page and selection. It is moved a little if it would be hard to read.">
    <Controller
      control={control}
      name="accent"
      render={({ field }) => (
        <div className="flex w-full items-center gap-2">
          <ColorPicker label="Accent" value={field.value} onChange={field.onChange} disabled={disabled} emptyLabel="Default" />
          {field.value !== "" && (
            <Button type="button" variant="ghost" size="sm" disabled={disabled} onClick={() => field.onChange("")}>
              Use the default
            </Button>
          )}
        </div>
      )}
    />
  </SettingsRow>
);
