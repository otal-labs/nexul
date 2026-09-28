import { SendHorizontal } from "lucide-react-native";
import { useState } from "react";
import { View } from "react-native";
import { useCSSVariable } from "uniwind";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { usePostMessage } from "@/hooks/ChatHooks";

interface ChatComposerProps {
  conversationId: string;
}

export const ChatComposer = ({ conversationId }: ChatComposerProps) => {
  const [draft, setDraft] = useState("");
  const post = usePostMessage(conversationId);
  const [primaryForeground] = useCSSVariable(["--color-primary-foreground"]);
  const canSend = draft.trim() !== "";

  // The draft clears at once because the optimistic row shows it; a failed send puts it back.
  const send = () => {
    const body = draft.trim();
    if (body === "") return;
    setDraft("");
    post.mutate(body, { onError: () => setDraft(body) });
  };

  return (
    <View className="gap-1 border-t border-border bg-background px-3 py-2">
      {post.error && <ErrorDisplay error={post.error} />}
      <View className="flex-row items-end gap-2">
        <Input
          multiline
          value={draft}
          onChangeText={setDraft}
          placeholder="Message"
          aria-label="Message"
          className="h-auto max-h-32 min-h-10 flex-1 py-2"
        />
        <Button size="icon" aria-label="Send" disabled={!canSend} onPress={send}>
          <SendHorizontal color={String(primaryForeground)} size={18} />
        </Button>
      </View>
    </View>
  );
};
