import { LegendList, type LegendListRef, type NativeScrollEvent, type NativeSyntheticEvent } from "@legendapp/list/react-native";
import { Stack, useLocalSearchParams } from "expo-router";
import { ArrowDown } from "lucide-react-native";
import { useRef, useState } from "react";
import { Pressable, View } from "react-native";
import { useCSSVariable } from "uniwind";

import { ContainerLogLineRow } from "@/components/deploys/ContainerLogLineRow";
import { LogsStatusBar } from "@/components/deploys/LogsStatusBar";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PlaceholderScreen } from "@/components/PlaceholderScreen";
import { Text } from "@/components/ui/text";
import { useContainerLogs } from "@/hooks/ContainerLogHooks";
import { useAreaAccess } from "@/hooks/WorkspaceHooks";

// Within this many points of the bottom still counts as following the tail.
const FOLLOW_SLACK = 48;

export const ContainerLogsScreen = () => {
  const { id, service } = useLocalSearchParams<{ id: string; service: string }>();
  const allowed = useAreaAccess()?.("stackLogs");
  const { lines, status, reason } = useContainerLogs(id, service, allowed === true);
  const listRef = useRef<LegendListRef>(null);
  const [following, setFollowing] = useState(true);
  const [mutedForeground] = useCSSVariable(["--color-muted-foreground"]);

  const onScroll = ({ nativeEvent }: NativeSyntheticEvent<NativeScrollEvent>) => {
    const { contentOffset, contentSize, layoutMeasurement } = nativeEvent;
    setFollowing(contentSize.height - contentOffset.y - layoutMeasurement.height < FOLLOW_SLACK);
  };

  const jumpToLive = () => {
    void listRef.current?.scrollToEnd({ animated: true });
    setFollowing(true);
  };

  const noOutput = lines.length === 0;

  return (
    <View className="flex-1 bg-background">
      <Stack.Screen options={{ title: service }} />
      {allowed === undefined && <LoadingDisplay />}
      {(allowed === false || status === "forbidden") && <PlaceholderScreen message="You can't read logs for this stack." />}
      {allowed === true && status !== "forbidden" && noOutput && status === "connecting" && (
        <LoadingDisplay message="Connecting…" />
      )}
      {allowed === true && noOutput && status === "offline" && (
        <PlaceholderScreen message={reason ? `Runner offline: ${reason}` : "Runner offline"} />
      )}
      {allowed === true && noOutput && (status === "live" || status === "ended") && <PlaceholderScreen message="No output yet" />}
      {allowed === true && !noOutput && status !== "forbidden" && (
        <>
          <LogsStatusBar status={status} following={following} />
          <View className="flex-1 bg-surface-2">
            <LegendList
              ref={listRef}
              data={lines}
              keyExtractor={(entry) => String(entry.id)}
              renderItem={({ item }) => <ContainerLogLineRow line={item} />}
              estimatedItemSize={20}
              onScroll={onScroll}
              recycleItems
              initialScrollAtEnd
              alignItemsAtEnd
              maintainScrollAtEnd
              maintainVisibleContentPosition
            />
            {!following && (
              <Pressable
                role="button"
                onPress={jumpToLive}
                className="absolute bottom-4 min-h-11 flex-row items-center gap-1.5 self-center rounded-full border border-border bg-card px-4 active:bg-accent"
              >
                <ArrowDown size={14} color={String(mutedForeground)} />
                <Text variant="small">Jump to live</Text>
              </Pressable>
            )}
          </View>
        </>
      )}
    </View>
  );
};
