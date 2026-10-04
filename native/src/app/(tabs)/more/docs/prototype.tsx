import { ClarifyProtoScreen } from "@/components/docs/prototype/ClarifyProtoScreen";

// Throwaway prototype route, dev builds only.
export default function ClarifyProtoRoute() {
  if (!__DEV__) return null;
  return <ClarifyProtoScreen />;
}
