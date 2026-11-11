import { Palette } from "lucide-react";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { InlineNotice } from "@/components/InlineNotice";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { GatedSaveBar } from "@/components/settings/GatedSaveBar";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { SettingsRows } from "@/components/settings/SettingsRow";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { BrandingAccentRow, BrandingNameRow } from "@/ee/components/BrandingFields";
import { BrandingLogoRow } from "@/ee/components/BrandingLogoRow";
import { BrandingResetButton } from "@/ee/components/BrandingResetButton";
import { BrandingThemeRows } from "@/ee/components/BrandingThemeRows";
import { useBrandingForm } from "@/ee/components/useBrandingForm";
import { useBranding } from "@/ee/hooks/BrandingHooks";
import type { Branding } from "@/ee/models/Branding";

const FORM_ID = "branding-form";

const BrandingStatus = ({ branding }: { branding: Branding }) => (
  <SettingsStatus tone={branding.unlocked ? "success" : "muted"}>{branding.unlocked ? "Unlocked" : "Needs a license key"}</SettingsStatus>
);

const hasAnything = (b: Branding): boolean => b.product_name !== "" || b.accent !== "" || !!b.logo || !!b.light_theme || !!b.dark_theme;

const BrandingBody = ({ branding }: { branding: Branding }) => {
  const { form, disabled, saved, onSubmit } = useBrandingForm(branding);
  return (
    <SettingsCard
      id="branding"
      title="Branding"
      description="Make this instance look like your own: its name, logo, accent and a VS Code color theme, for everyone on it."
      icon={Palette}
      aside={<BrandingStatus branding={branding} />}
      footer={
        <>
          {hasAnything(branding) && <BrandingResetButton />}
          <GatedSaveBar
            permission="instance:write"
            verb="change the branding"
            form={FORM_ID}
            dirty={form.formState.isDirty}
            saving={form.formState.isSubmitting}
            saved={saved}
            onDiscard={() => form.reset()}
            hint="The logo and themes save when chosen."
          />
        </>
      }
    >
      {!branding.unlocked && (
        <InlineNotice title="Branding needs a license key" tone="warning" className="mb-4">
          What is saved here stays, and shows again once <a href="#license" className="underline underline-offset-4">License</a> holds a key that includes Branding.
        </InlineNotice>
      )}
      <form id={FORM_ID} key={branding.product_name + branding.accent} onSubmit={onSubmit}>
        <SettingsRows>
          <BrandingNameRow control={form.control} disabled={disabled} />
          <BrandingLogoRow branding={branding} />
          <BrandingAccentRow control={form.control} disabled={disabled} />
          <BrandingThemeRows branding={branding} />
        </SettingsRows>
      </form>
    </SettingsCard>
  );
};

export const BrandingSection = () => {
  const { data, isPending, error } = useBranding();
  return (
    <>
      {isPending && (
        <SettingsCard id="branding" title="Branding" icon={Palette}>
          <LoadingDisplay />
        </SettingsCard>
      )}
      {error && (
        <SettingsCard id="branding" title="Branding" icon={Palette}>
          <ErrorDisplay error={error} />
        </SettingsCard>
      )}
      {data && <BrandingBody branding={data} />}
    </>
  );
};

