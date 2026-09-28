import { useLocalSearchParams } from "expo-router";

import { DocScreen } from "@/components/docs/DocScreen";

export default function DocRoute() {
  const { id } = useLocalSearchParams<{ id: string }>();
  return <DocScreen docId={id ?? ""} />;
}
