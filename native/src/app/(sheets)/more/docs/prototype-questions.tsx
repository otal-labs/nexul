import { ClarifyProtoSheet } from "@/components/docs/prototype/ClarifyProtoVariantC";

// Throwaway prototype sheet, dev builds only.
export default function ClarifyProtoSheetRoute() {
  if (!__DEV__) return null;
  return <ClarifyProtoSheet />;
}
