import { Stack, useLocalSearchParams } from "expo-router";
import { ScrollView, View } from "react-native";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { MessageBody } from "@/components/chat/MessageBody";
import { useFetchNoteMarkdown } from "@/hooks/AttachmentHooks";

export const NoteScreen = () => {
  const { id, name } = useLocalSearchParams<{ id: string; name?: string }>();
  const { data: markdown, error, isPending } = useFetchNoteMarkdown(id);
  return (
    <View className="flex-1 bg-background">
      <Stack.Screen options={{ title: name ?? "Note" }} />
      {isPending && <LoadingDisplay message="Loading the note" />}
      {error && <ErrorDisplay error={error} notFound="This note doesn't exist or was deleted." />}
      {markdown !== undefined && (
        <ScrollView contentContainerClassName="p-3 pb-8">
          <View className="gap-3 rounded-xl border border-border bg-card px-4 py-5">
            <MessageBody body={markdown} />
          </View>
        </ScrollView>
      )}
    </View>
  );
};
