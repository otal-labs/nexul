import { View } from "react-native";

import { ClarificationRoundsFeed } from "@/components/docs/clarification/ClarificationRoundsFeed";
import { ClarificationStatusLine } from "@/components/docs/clarification/ClarificationStatusLine";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Text } from "@/components/ui/text";
import { useFetchDocClarification } from "@/hooks/DocHooks";
import { useAreaAccess } from "@/hooks/WorkspaceHooks";
import type { Doc } from "@/models/Doc";

interface DocQuestionsPanelProps {
  doc: Doc;
}

// The Questions side, in the body's place; a locked doc still takes answers, and a reader sees them read-only.
export const DocQuestionsPanel = ({ doc }: DocQuestionsPanelProps) => {
  const { data: clarification, error, isPending } = useFetchDocClarification(doc.id);
  const canWrite = useAreaAccess(doc.project_id)?.("editDocs") ?? false;
  return (
    <View className="px-5">
      {isPending && <LoadingDisplay message="Loading the questions" />}
      {error && <ErrorDisplay error={error} />}
      {clarification && (
        <View>
          <Text role="heading" className="text-lg font-semibold leading-snug tracking-tight">
            Questions
          </Text>
          <View className="mt-1">
            <ClarificationStatusLine clarification={clarification} locked={doc.locked} />
          </View>
          <ClarificationRoundsFeed clarification={clarification} readOnly={!canWrite} />
        </View>
      )}
    </View>
  );
};
