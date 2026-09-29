import { useNavigation } from "expo-router";
import { useEffect } from "react";
import { ScrollView, View } from "react-native";

import { DocBody } from "@/components/docs/DocBody";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Text } from "@/components/ui/text";
import { useFetchDoc } from "@/hooks/DocHooks";
import { RelativeTime } from "@/components/RelativeTime";

interface DocScreenProps {
  docId: string;
}

export const DocScreen = ({ docId }: DocScreenProps) => {
  const navigation = useNavigation();
  const { data: doc, error, isPending } = useFetchDoc(docId);

  useEffect(() => {
    navigation.setOptions({ title: doc?.title ?? "Doc" });
  }, [navigation, doc]);

  return (
    <View className="flex-1 bg-background">
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} notFound="This doc doesn't exist or was deleted." />}
      {doc && (
        <ScrollView contentContainerClassName="gap-3 p-4">
          <Text variant="h3">{doc.title}</Text>
          <Text variant="muted" className="font-mono text-xs">
            v{doc.version} · updated <RelativeTime iso={doc.updated_at} />
          </Text>
          <DocBody body={doc.body} />
        </ScrollView>
      )}
    </View>
  );
};
