import { CornerDownRight } from "lucide-react-native";
import { View } from "react-native";

import { Icon } from "@/components/ui/icon";
import { Text } from "@/components/ui/text";
import { Textarea } from "@/components/ui/textarea";
import { ROUNDS } from "@/components/docs/prototype/ClarifyProtoData";
import { useClarifyProtoStore } from "@/components/docs/prototype/ClarifyProtoStore";

interface ClarifyProtoNoteProps {
  round: number;
}

interface ClarifyProtoNoteBoxProps extends ClarifyProtoNoteProps {
  // On its own page the box is the page's question, so it reads at the question's size.
  large?: boolean;
}

// The round's closing "Anything else?" box while the round is the newest one.
export const ClarifyProtoNoteBox = ({ round, large = false }: ClarifyProtoNoteBoxProps) => {
  const note = useClarifyProtoStore((s) => s.notes[round] ?? "");
  const setNote = useClarifyProtoStore((s) => s.setNote);
  return (
    <View className={large ? "" : "pt-4"}>
      <Text className={large ? "text-xl font-semibold tracking-tight" : "font-medium"}>Anything else?</Text>
      <Text className="text-sm text-muted-foreground leading-snug">{"Optional. A question of your own, or something these didn't cover."}</Text>
      <Textarea
        aria-label="Anything else?"
        value={note}
        onChangeText={(text) => setNote(round, text)}
        placeholder="Type it here"
        numberOfLines={6}
        className={large ? "mt-4 min-h-32" : "mt-2 min-h-20"}
      />
    </View>
  );
};

// A past round's "Anything else?" as asked, with the next round's one-line reply under it.
export const ClarifyProtoNoteReply = ({ round }: ClarifyProtoNoteProps) => {
  const note = useClarifyProtoStore((s) => s.notes[round] ?? "");
  const answered = useClarifyProtoStore((s) => round < s.rounds || s.phase === "closed");
  const reply = answered ? (ROUNDS[round - 1]?.reply ?? "") : "";
  if (note.trim() === "") return null;
  return (
    <View className="mt-2 gap-1">
      <Text className="font-mono text-xs text-muted-foreground leading-snug">Anything else?</Text>
      <Text className="text-sm text-muted-foreground leading-snug">{note}</Text>
      {reply !== "" && (
        <View className="flex-row items-start gap-1.5">
          <Icon as={CornerDownRight} size={14} className="mt-0.5 text-muted-foreground" />
          <Text className="min-w-0 flex-1 text-sm leading-snug">{reply}</Text>
        </View>
      )}
    </View>
  );
};
