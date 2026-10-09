import { useRouter } from "expo-router";
import FileText from "lucide-react-native/icons/file-text";
import { Pressable, View } from "react-native";
import { useCSSVariable } from "uniwind";

import { Text } from "@/components/ui/text";
import { useFetchConversationFiles } from "@/hooks/AttachmentHooks";
import { formatBytes } from "@/models/Attachment";

interface NoteFilePillProps {
  conversationId: string;
  attachmentId: string;
}

// A note's file, found among its thread's files; it opens the read-only render, editing stays on the web.
export const NoteFilePill = ({ conversationId, attachmentId }: NoteFilePillProps) => {
  const router = useRouter();
  const [mutedForeground] = useCSSVariable(["--color-muted-foreground"]);
  const { data: files } = useFetchConversationFiles(conversationId);
  const file = files?.find((f) => f.id === attachmentId);
  if (!file) return null;
  return (
    <Pressable
      role="button"
      aria-label={`Open ${file.name}`}
      onPress={() => router.push({ pathname: "/chat/note/[id]", params: { id: file.id, name: file.name } })}
      className="min-h-11 flex-row items-center self-start"
    >
      <View className="max-w-full flex-row items-center gap-1.5 rounded-full border border-border bg-card px-2.5 py-1 active:bg-accent">
        <FileText color={String(mutedForeground)} size={14} />
        <Text numberOfLines={1} className="shrink text-xs">
          {file.name}
        </Text>
        <Text className="font-mono text-[11px] text-muted-foreground">{formatBytes(file.size)}</Text>
      </View>
    </Pressable>
  );
};
