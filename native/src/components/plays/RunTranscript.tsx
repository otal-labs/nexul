import { View } from "react-native";

import { segmentTranscript } from "@nexul/client-core/trail";

import { HandoffTurnGroup } from "@/components/chat/HandoffTurnGroup";
import { MessageBody } from "@/components/chat/MessageBody";
import { RunQuestionCard } from "@/components/plays/RunQuestionCard";
import { Text } from "@/components/ui/text";
import type { Trail } from "@/models/Trail";

// The starter's words as a bubble on the right, as the web's transcript draws them.
const StarterBubble = ({ body }: { body: string }) => (
  <View className="max-w-[85%] self-end rounded-xl border border-border bg-card px-3.5 py-2.5">
    <MessageBody body={body} />
  </View>
);

// The run as a conversation: the starter's bubbles, the Agent's turns folded, the question
// where it was asked, the reply as prose, and the runner's own lines muted.
export const RunTranscript = ({ trail }: { trail: Trail }) => (
  <View className="gap-3">
    {segmentTranscript(trail, trail.activity, trail.state, trail.question ?? null).map((segment, i) => (
      <View key={`${segment.kind}-${i}`}>
        {segment.kind === "user" && <StarterBubble body={segment.body} />}
        {segment.kind === "turn" && <HandoffTurnGroup key={String(segment.running)} entries={segment.entries} running={segment.running} from={segment.from} until={segment.until} />}
        {segment.kind === "question" && <RunQuestionCard trail={trail} />}
        {segment.kind === "reply" && <MessageBody body={segment.entry.detail || segment.entry.summary} />}
        {segment.kind === "note" && <Text className="px-1 text-row text-muted-foreground">{segment.text}</Text>}
      </View>
    ))}
  </View>
);
