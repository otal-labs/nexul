import { LockIcon } from "lucide-react-native";
import { View } from "react-native";

import { Icon } from "@/components/ui/icon";
import { Text } from "@/components/ui/text";
import { MessageBody } from "@/components/chat/MessageBody";
import { DOC_AFTER, DOC_BEFORE, DOC_TITLE } from "@/components/docs/prototype/ClarifyProtoData";
import { useClarifyProtoStore } from "@/components/docs/prototype/ClarifyProtoStore";

// The doc screen's title and meta line as DocScreen draws them, with the plain locked state while a round is written.
export const ClarifyProtoDocHeader = () => {
  const locked = useClarifyProtoStore((s) => s.phase === "running");
  return (
    <View className="gap-1">
      <Text variant="h3" className="leading-tight">
        {DOC_TITLE}
      </Text>
      <View className="flex-row flex-wrap items-center gap-x-2">
        <Text variant="muted" className="font-mono text-xs leading-snug">
          v3 · updated 2h ago
        </Text>
        {locked && (
          <View className="flex-row items-center gap-1">
            <Icon as={LockIcon} size={12} className="text-muted-foreground" />
            <Text variant="muted" className="font-mono text-xs leading-snug">
              Locked
            </Text>
          </View>
        )}
      </View>
    </View>
  );
};

// The body: the client's words until the clarification closes, then the doc written from every answer.
export const ClarifyProtoDocBody = () => {
  const written = useClarifyProtoStore((s) => s.phase === "closed");
  return <MessageBody body={written ? DOC_AFTER : DOC_BEFORE} />;
};
