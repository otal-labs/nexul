import type { ClarificationRound } from "@nexul/client-core/clarification";
import CornerDownRight from "lucide-react-native/icons/corner-down-right";
import { View } from "react-native";

import { Icon } from "@/components/ui/icon";
import { Text } from "@/components/ui/text";

interface AnythingElseReplyProps {
  round: ClarificationRound;
}

// A round's "Anything else?" as written, with the next round's one-line reply under it; shown even while the round is folded.
export const AnythingElseReply = ({ round }: AnythingElseReplyProps) => (
  <>
    {round.anything_else !== "" && (
      <View className="mt-2 gap-1">
        <Text className="font-mono text-xs leading-snug text-muted-foreground">Anything else?</Text>
        <Text className="text-sm leading-snug text-muted-foreground">{round.anything_else}</Text>
        {round.anything_else_reply !== "" && (
          <View className="flex-row items-start gap-1.5">
            <Icon as={CornerDownRight} size={14} className="mt-0.5 text-muted-foreground" />
            <Text className="min-w-0 flex-1 text-sm leading-snug">{round.anything_else_reply}</Text>
          </View>
        )}
      </View>
    )}
  </>
);
