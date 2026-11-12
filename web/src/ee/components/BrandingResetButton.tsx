import { RotateCcw } from "lucide-react";

import { ConfirmDestroyButton } from "@/components/settings/ConfirmDestroyButton";
import { useClearBranding } from "@/ee/hooks/BrandingHooks";

export const BrandingResetButton = () => {
  const clear = useClearBranding();
  return (
    <ConfirmDestroyButton
      icon={RotateCcw}
      idleLabel="Remove branding"
      confirmLabel="Remove"
      labelled
      loading={clear.isPending}
      onConfirm={() => clear.mutate()}
      gate={{ permission: "instance:write", verb: "remove the branding" }}
    />
  );
};
