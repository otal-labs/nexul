import { toast } from "sonner";

import { ActionButton } from "@/components/action/ActionButton";
import { SettingsRow } from "@/components/settings/SettingsRow";
import { BrandingFileButton } from "@/ee/components/BrandingFileButton";
import { BrandingLogoPreview } from "@/ee/components/BrandingLogoPreview";
import { useRemoveBrandingLogo, useSetBrandingLogo } from "@/ee/hooks/BrandingHooks";
import { MAX_LOGO_BYTES, type Branding } from "@/ee/models/Branding";
import { isOffSquare, measureLogo } from "@/ee/utils/LogoShape";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";

const LOGO_TYPES = ["image/png", "image/jpeg", "image/webp", "image/svg+xml"];

export const BrandingLogoRow = ({ branding }: { branding: Branding }) => {
  const set = useSetBrandingLogo();
  const remove = useRemoveBrandingLogo();
  const { open: confirm } = useConfirmationDialog();
  const locked = !branding.unlocked;
  // The image is served to the page only while a key unlocks branding, so a locked instance shows an empty tile and says one is saved.
  const shows = !!branding.logo && !locked;

  const confirmShape = async (file: File): Promise<boolean> => {
    const size = await measureLogo(file);
    if (!size || !isOffSquare(size)) return true;
    const url = URL.createObjectURL(file);
    const ok = await confirm({
      title: "This logo isn't square",
      message: "The mark is a square, so a wide or tall image shrinks to fit it and its lettering gets small. This is how it will look.",
      details: <BrandingLogoPreview src={url} />,
      confirmLabel: "Use it anyway",
      cancelLabel: "Choose another",
      destructive: false,
    });
    URL.revokeObjectURL(url);
    return ok;
  };

  const choose = async (file: File) => {
    if (!LOGO_TYPES.includes(file.type)) return void toast.error("A logo is a PNG, JPEG, WebP or SVG image");
    if (file.size > MAX_LOGO_BYTES) return void toast.error("A logo is at most 512 KB");
    if (await confirmShape(file)) set.mutate(file);
  };

  return (
    <SettingsRow label="Logo" description="Replaces the Nexul mark in the sidebar and on sign-in, and is the tab's icon. Square images fit best. Saved when you choose it.">
      <div className="flex w-full flex-col gap-2">
        <div className="flex items-center gap-3">
          <span className="grid size-10 shrink-0 place-items-center overflow-hidden rounded-lg bg-surface-2 ring-1 ring-border">
            {shows && <img src={`/api/branding/logo?v=${branding.logo?.version ?? ""}`} alt="Current logo" className="size-full object-contain" />}
            {!branding.logo && <span className="text-xs text-muted-foreground">None</span>}
          </span>
          <div className="flex flex-wrap gap-x-1 gap-y-4">
            <BrandingFileButton
              label={branding.logo ? "Replace" : "Upload"}
              accept={LOGO_TYPES.join(",")}
              verb="change the logo"
              disabled={locked}
              busy={set.isPending}
              onFile={(f) => void choose(f)}
            />
            {branding.logo && (
              <ActionButton type="button" permission="instance:write" verb="remove the logo" size="sm" variant="ghost" loading={remove.isPending} onClick={() => remove.mutate()}>
                Remove
              </ActionButton>
            )}
          </div>
        </div>
        {branding.logo && locked && <p className="text-xs text-muted-foreground">Saved, shows once licensed.</p>}
      </div>
    </SettingsRow>
  );
};
