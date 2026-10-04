import { View } from "react-native";
import { useShallow } from "zustand/react/shallow";

import { Text } from "@/components/ui/text";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { ClarifyProtoDocBody, ClarifyProtoDocHeader } from "@/components/docs/prototype/ClarifyProtoDoc";
import { ClarifyProtoPanel } from "@/components/docs/prototype/ClarifyProtoStatus";
import { pendingCount, useClarifyProtoStore, type ProtoTab } from "@/components/docs/prototype/ClarifyProtoStore";

// The web pick on a phone: a full-width "Doc | Questions N" switch under the title; Questions takes the body's place.
const ViewSwitch = () => {
  const { tab, setTab, pending } = useClarifyProtoStore(
    useShallow((s) => ({ tab: s.tab, setTab: s.setTab, pending: pendingCount(s) })),
  );
  return (
    <ToggleGroup
      type="single"
      variant="outline"
      value={tab}
      onValueChange={(value) => value && setTab(value as ProtoTab)}
      aria-label="Doc or questions"
      className="w-full"
    >
      <ToggleGroupItem value="doc" isFirst className="h-12 flex-1">
        <Text>Doc</Text>
      </ToggleGroupItem>
      <ToggleGroupItem value="questions" isLast className="h-12 flex-1">
        <Text>Questions</Text>
        {pending > 0 && <Text className="font-mono text-xs text-muted-foreground leading-snug">{pending}</Text>}
      </ToggleGroupItem>
    </ToggleGroup>
  );
};

export const ClarifyProtoVariantA = () => {
  const tab = useClarifyProtoStore((s) => s.tab);
  return (
    <View className="gap-4">
      <ClarifyProtoDocHeader />
      <ViewSwitch />
      {tab === "doc" && <ClarifyProtoDocBody />}
      {tab === "questions" && <ClarifyProtoPanel />}
    </View>
  );
};
