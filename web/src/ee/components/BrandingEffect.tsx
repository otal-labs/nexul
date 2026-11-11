import { useApplyBranding } from "@/ee/hooks/useApplyBranding";

// Paints the instance's look onto the page; it draws nothing itself.
export const BrandingEffect = () => {
  useApplyBranding();
  return null;
};
