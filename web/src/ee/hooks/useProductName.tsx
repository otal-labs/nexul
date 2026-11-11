import { useLook } from "@/ee/hooks/BrandingHooks";
import { productName } from "@/ee/models/Branding";

// What the instance calls itself: the Owner's product name while a key unlocks branding, otherwise Nexul.
export const useProductName = (): string => productName(useLook().data);
