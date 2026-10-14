import { Stack } from "expo-router";
import Lock from "lucide-react-native/icons/lock";
import { ScrollView, View } from "react-native";

import { DocBody } from "@/components/docs/DocBody";
import { DocQuestionsPanel } from "@/components/docs/clarification/DocQuestionsPanel";
import { DocViewSwitch } from "@/components/docs/clarification/DocViewSwitch";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ProjectRevokedGate } from "@/components/project/ProjectRevokedGate";
import { RelativeTime } from "@/components/RelativeTime";
import { ScreenHeader } from "@/components/ScreenHeader";
import { Icon } from "@/components/ui/icon";
import { Text } from "@/components/ui/text";
import { useFetchDoc } from "@/hooks/DocHooks";
import { useDocView, type DocView } from "@/hooks/useDocView";
import type { Doc } from "@/models/Doc";

interface DocScreenProps {
  docId: string;
  // A notice about questions opens the screen on them.
  openedOn?: DocView;
}

const DocMeta = ({ doc }: { doc: Doc }) => (
  <>
    <Text className="font-mono text-xs text-muted-foreground">
      v{doc.version} · updated <RelativeTime iso={doc.updated_at} />
    </Text>
    {doc.locked && (
      <View className="flex-row items-center gap-1">
        <Icon as={Lock} size={12} className="text-muted-foreground" />
        <Text className="font-mono text-xs text-muted-foreground">Locked</Text>
      </View>
    )}
  </>
);

const DocSides = ({ doc, openedOn }: { doc: Doc; openedOn: DocView }) => {
  const { switchable, waiting, view, setView } = useDocView(doc.id, openedOn);
  return (
    <>
      <ScreenHeader title={doc.title} meta={<DocMeta doc={doc} />} className="pt-2" />
      {switchable && <DocViewSwitch view={view} waiting={waiting} onChange={setView} />}
      {view === "doc" && (
        <View className="mx-3 rounded-xl border border-border bg-card px-4 py-5">
          <DocBody body={doc.body} />
        </View>
      )}
      {view === "questions" && <DocQuestionsPanel doc={doc} />}
    </>
  );
};

// The title stays out of the sheet; the body is a sheet on the canvas, the way the web lays a doc out.
export const DocScreen = ({ docId, openedOn = "doc" }: DocScreenProps) => {
  const { data: doc, error, isPending } = useFetchDoc(docId);

  return (
    <ProjectRevokedGate projectId={doc?.project_id}>
      <View className="flex-1 bg-background">
        <Stack.Screen options={{ title: "Doc" }} />
        {isPending && <LoadingDisplay message="Loading the doc" />}
        {error && <ErrorDisplay error={error} notFound="This doc doesn't exist or was deleted." />}
        {doc && (
          <ScrollView contentContainerClassName="pb-8" keyboardShouldPersistTaps="handled">
            <DocSides doc={doc} openedOn={openedOn} />
          </ScrollView>
        )}
      </View>
    </ProjectRevokedGate>
  );
};
