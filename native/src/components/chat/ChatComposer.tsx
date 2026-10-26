import ArrowUp from "lucide-react-native/icons/arrow-up";
import Check from "lucide-react-native/icons/check";
import { useState } from "react";
import { Pressable, TextInput, View } from "react-native";
import { useCSSVariable } from "uniwind";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { Text } from "@/components/ui/text";
import { useEditMessage, usePostMessage } from "@/hooks/ChatHooks";
import { cn } from "@/lib/utils";
import { useMessageEditStore, type EditingMessage } from "@/stores/messageEditStore";

interface ChatComposerProps {
  conversationId: string;
  // Mounted with key={editing?.messageId}: the draft seeds once, so entering or leaving an edit must remount.
  editing?: EditingMessage | null;
}

// One framed field holding the text and the send button; the frame turns to the ink focus ring, and Send is the ember once
// there is something to send.
export const ChatComposer = ({ conversationId, editing = null }: ChatComposerProps) => {
  const [draft, setDraft] = useState(editing?.body ?? "");
  const [focused, setFocused] = useState(false);
  const post = usePostMessage(conversationId);
  const edit = useEditMessage();
  const stopEditing = useMessageEditStore((s) => s.stop);
  const [brandForeground, muted] = useCSSVariable(["--color-brand-foreground", "--color-muted-foreground"]);
  const error = post.error ?? edit.error;
  const canSend = draft.trim() !== "" && draft.trim() !== editing?.body;

  // The draft clears at once because the optimistic row shows it; a failed send puts it back.
  const send = () => {
    const body = draft.trim();
    if (body === "") return;
    if (editing) {
      edit.mutate({ messageId: editing.messageId, body }, { onSuccess: stopEditing });
      return;
    }
    setDraft("");
    post.mutate(body, { onError: () => setDraft(body) });
  };

  return (
    <View className="gap-1 bg-background px-3 pb-2 pt-1.5">
      {editing && (
        <View className="flex-row items-center justify-between px-1">
          <Text className="text-[13px] font-medium text-muted-foreground">Editing your message</Text>
          <Pressable role="button" aria-label="Cancel editing" onPress={stopEditing} hitSlop={8} className="min-h-8 justify-center px-1">
            <Text className="text-[13px] font-medium">Cancel</Text>
          </Pressable>
        </View>
      )}
      {error && <ErrorDisplay error={error} className="px-1" />}
      <View className={cn("flex-row items-end rounded-lg border bg-card py-1 pl-3.5 pr-1", focused ? "border-focus" : "border-input")}>
        <TextInput
          multiline
          autoFocus={!!editing}
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
          aria-label={editing ? "Save" : "Send"}
          disabled={!canSend}
          onPress={send}
          className={cn("size-10 items-center justify-center rounded-md", canSend && "bg-brand active:bg-brand/85")}
        >
          {editing && <Check color={String(canSend ? brandForeground : muted)} size={20} />}
          {!editing && <ArrowUp color={String(canSend ? brandForeground : muted)} size={20} />}
        </Pressable>
      </View>
    </View>
  );
};
