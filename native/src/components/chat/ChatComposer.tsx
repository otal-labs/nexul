import ArrowUp from "lucide-react-native/icons/arrow-up";
import { useState } from "react";
import { Pressable, TextInput, View } from "react-native";
import { useCSSVariable } from "uniwind";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { usePostMessage } from "@/hooks/ChatHooks";
import { cn } from "@/lib/utils";

interface ChatComposerProps {
  conversationId: string;
}

// One framed field holding the text and the send button; the frame turns to the ink focus ring, and Send is the ember once
// there is something to send.
export const ChatComposer = ({ conversationId }: ChatComposerProps) => {
  const [draft, setDraft] = useState("");
  const [focused, setFocused] = useState(false);
  const post = usePostMessage(conversationId);
  const [brandForeground, muted] = useCSSVariable(["--color-brand-foreground", "--color-muted-foreground"]);
  const canSend = draft.trim() !== "";

  // The draft clears at once because the optimistic row shows it; a failed send puts it back.
  const send = () => {
    const body = draft.trim();
    if (body === "") return;
    setDraft("");
    post.mutate(body, { onError: () => setDraft(body) });
  };

  return (
    <View className="gap-1 bg-background px-3 pb-2 pt-1.5">
      {post.error && <ErrorDisplay error={post.error} className="px-1" />}
      <View className={cn("flex-row items-end rounded-xl border bg-card py-1 pl-3.5 pr-1", focused ? "border-focus" : "border-input")}>
        <TextInput
          multiline
          value={draft}
          onChangeText={setDraft}
          onFocus={() => setFocused(true)}
          onBlur={() => setFocused(false)}
          placeholder="Message"
          aria-label="Message"
          placeholderTextColorClassName="accent-muted-foreground"
          className="max-h-32 min-h-10 flex-1 py-2.5 font-sans text-base leading-5 text-foreground"
        />
        <Pressable
          role="button"
          aria-label="Send"
          disabled={!canSend}
          onPress={send}
          className={cn("size-10 items-center justify-center rounded-lg", canSend && "bg-brand active:bg-brand/85")}
        >
          <ArrowUp color={String(canSend ? brandForeground : muted)} size={20} />
        </Pressable>
      </View>
    </View>
  );
};
