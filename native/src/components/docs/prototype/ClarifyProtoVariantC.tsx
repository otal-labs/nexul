import { useRouter } from "expo-router";
import { LockIcon } from "lucide-react-native";
import { ScrollView, View } from "react-native";

import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icon";
import { Text } from "@/components/ui/text";
import { ClarifyProtoDocBody, ClarifyProtoDocHeader } from "@/components/docs/prototype/ClarifyProtoDoc";
import { ClarifyProtoPanel, ClarifyProtoStatusIcon, useStatusLine } from "@/components/docs/prototype/ClarifyProtoStatus";

export const ClarifyProtoVariantC = () => (
  <View className="gap-4">
    <ClarifyProtoDocHeader />
    <ClarifyProtoDocBody />
  </View>
);

// C's bar pinned under the doc, above the tab bar: the state line and the button that lifts the round up in a sheet.
export const ClarifyProtoBottomBar = () => {
  const router = useRouter();
  const { icon, label, detail } = useStatusLine();
  const waiting = icon === "waiting";
  return (
    <View className="flex-row items-center gap-3 border-t border-border bg-card px-4 py-2">
      <View className="min-w-0 flex-1 py-1">
        <View className="flex-row items-center gap-2">
          {icon === null && <Icon as={LockIcon} size={14} className="text-muted-foreground" />}
          <ClarifyProtoStatusIcon icon={icon} />
          <Text className={waiting ? "min-w-0 flex-1 font-semibold" : "min-w-0 flex-1 text-muted-foreground"}>{label}</Text>
        </View>
        {detail !== "" && (
          <Text numberOfLines={1} className="text-sm text-muted-foreground leading-snug">
            {detail}
          </Text>
        )}
      </View>
      <Button
        variant={waiting ? "default" : "outline"}
        onPress={() => router.push("/more/docs/prototype-questions")}
        className="h-12 px-5"
      >
        <Text>{waiting ? "Answer" : "View"}</Text>
      </Button>
    </View>
  );
};

// The round over the doc: the same Questions checklist as the tab, in a sheet the system back closes.
export const ClarifyProtoSheet = () => (
  <ScrollView className="bg-popover" contentContainerClassName="px-4 pb-10 pt-6" keyboardShouldPersistTaps="handled" nestedScrollEnabled>
    <ClarifyProtoPanel />
  </ScrollView>
);
