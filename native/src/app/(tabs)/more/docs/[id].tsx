import { useLocalSearchParams } from "expo-router";

import { DocScreen } from "@/components/docs/DocScreen";

export default function DocRoute() {
  const { id, view } = useLocalSearchParams<{ id: string; view?: string }>();
  return <DocScreen docId={id ?? ""} openedOn={view === "questions" ? "questions" : "doc"} />;
}
