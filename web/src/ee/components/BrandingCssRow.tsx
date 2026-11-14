import { Controller, type Control } from "react-hook-form";

import { SettingsRow } from "@/components/settings/SettingsRow";
import { Textarea } from "@/components/ui/textarea";
import { MAX_CSS_CHARS, type BrandingFormData } from "@/ee/models/Branding";

type CssRowProps = { control: Control<BrandingFormData>; disabled: boolean };

// Rules added after the colours on every page; an agent can write them through instance_update as well.
export const BrandingCssRow = ({ control, disabled }: CssRowProps) => (
  <SettingsRow
    label="Custom CSS"
    description="Your own style rules on every page. Images go inline as data: addresses; nothing loads from elsewhere. Or ask an agent to restyle the instance."
    htmlFor="branding-css-field"
  >
    <Controller
      control={control}
      name="css"
      render={({ field, fieldState }) => (
        <div className="w-full space-y-1">
          <Textarea
            {...field}
            id="branding-css-field"
            disabled={disabled}
            spellCheck={false}
            rows={6}
            maxLength={MAX_CSS_CHARS}
            placeholder={":root.dark { --brand: #7c5cff !important; }"}
            aria-invalid={!!fieldState.error}
            className="font-mono text-xs"
          />
          {fieldState.error && <p className="text-xs text-destructive">{fieldState.error.message}</p>}
        </div>
      )}
    />
  </SettingsRow>
);
