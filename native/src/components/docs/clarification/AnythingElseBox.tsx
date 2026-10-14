import type { ClarificationRound } from "@nexul/client-core/clarification";
import { useState } from "react";
import { View } from "react-native";

import { errorMessage } from "@/api/errors";
import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import { Textarea } from "@/components/ui/textarea";
import { useSaveAnythingElse } from "@/hooks/DocHooks";

interface AnythingElseBoxProps {
  round: ClarificationRound;
}

const AnythingElseField = ({ round }: AnythingElseBoxProps) => {
  const save = useSaveAnythingElse(round.doc_id);
  const [text, setText] = useState(round.anything_else);
  const changed = text.trim() !== round.anything_else;
  const submit = () => changed && save.mutate({ round: round.round, text });
  return (
    <View className="pt-4">
      <Text className="font-medium">Anything else?</Text>
      <Text className="text-sm leading-snug text-muted-foreground">{"Optional. A question of your own, or something these didn't cover."}</Text>
      <Textarea
        aria-label="Anything else?"
        value={text}
        onChangeText={setText}
        onBlur={submit}
        editable={!save.isPending}
        placeholder="Type it here"
        numberOfLines={6}
        className="mt-2 min-h-20"
      />
      {save.error && <Text role="alert" className="mt-2 text-sm leading-snug text-destructive">{errorMessage(save.error)}</Text>}
      {changed && (
        <Button variant="outline" size="sm" disabled={save.isPending} onPress={submit} className="mt-2 self-end">
          <Text>Save</Text>
        </Button>
      )}
    </View>
  );
};

// Saved on blur or Save; remounts on someone else's saved text so the box shows it.
export const AnythingElseBox = ({ round }: AnythingElseBoxProps) => <AnythingElseField key={round.anything_else} round={round} />;
