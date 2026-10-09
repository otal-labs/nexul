import { Stack } from "expo-router";
import { ScrollView, View } from "react-native";

import { DocBody } from "@/components/docs/DocBody";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ProjectRevokedGate } from "@/components/project/ProjectRevokedGate";
import { RelativeTime } from "@/components/RelativeTime";
import { ScreenHeader } from "@/components/ScreenHeader";
import { Text } from "@/components/ui/text";
import { useFetchDoc } from "@/hooks/DocHooks";

interface DocScreenProps {
  docId: string;
}

const DocMeta = ({ version, updatedAt }: { version: number; updatedAt: string }) => (
  <Text className="font-mono text-xs text-muted-foreground">
    v{version} · updated <RelativeTime iso={updatedAt} />
  </Text>
);

// The title stays out of the sheet; the body is a sheet on the canvas, the way the web lays a doc out.
export const DocScreen = ({ docId }: DocScreenProps) => {
  const { data: doc, error, isPending } = useFetchDoc(docId);

  return (
    <ProjectRevokedGate projectId={doc?.project_id}>
      <View className="flex-1 bg-background">
        <Stack.Screen options={{ title: "Doc" }} />
        {isPending && <LoadingDisplay message="Loading the doc" />}
        {error && <ErrorDisplay error={error} notFound="This doc doesn't exist or was deleted." />}
        {doc && (
          <ScrollView contentContainerClassName="pb-8">
            <ScreenHeader title={doc.title} meta={<DocMeta version={doc.version} updatedAt={doc.updated_at} />} className="pt-2" />
            <View className="mx-3 rounded-xl border border-border bg-card px-4 py-5">
              <DocBody body={doc.body} />
            </View>
          </ScrollView>
        )}
      </View>
    </ProjectRevokedGate>
  );
};
