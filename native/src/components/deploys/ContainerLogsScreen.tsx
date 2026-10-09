import { LegendList, type LegendListRef, type NativeScrollEvent, type NativeSyntheticEvent } from "@legendapp/list/react-native";
import { Stack, useLocalSearchParams } from "expo-router";
import ArrowDown from "lucide-react-native/icons/arrow-down";
import ScrollText from "lucide-react-native/icons/scroll-text";
import WifiOff from "lucide-react-native/icons/wifi-off";
import { useRef, useState } from "react";
import { Pressable, View } from "react-native";
import { useCSSVariable } from "uniwind";

import { ContainerLogLineRow } from "@/components/deploys/ContainerLogLineRow";
import { LogsStatusBar } from "@/components/deploys/LogsStatusBar";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { EmptyState } from "@/components/EmptyState";
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
      {(allowed === false || status === "forbidden") && <EmptyState size="compact" icon={ScrollText} title="No access to these logs" message="Reading a stack's logs needs the stack logs permission." />}
      {allowed === true && status !== "forbidden" && noOutput && status === "connecting" && (
        <LoadingDisplay message="Connecting" />
      )}
      {allowed === true && noOutput && status === "offline" && (
        <EmptyState size="compact" icon={WifiOff} title="Runner offline" message={reason || "The logs resume once its runner reconnects."} />
      )}
      {allowed === true && noOutput && (status === "live" || status === "ended") && <EmptyState size="compact" icon={ScrollText} title="No output yet" message="Lines the service prints show up here as they arrive." />}
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
                className="absolute bottom-4 min-h-11 flex-row items-center gap-1.5 self-center rounded-md border border-border bg-popover px-4 active:bg-accent"
              >
                <ArrowDown size={14} color={String(mutedForeground)} />
                <Text className="text-sm font-medium">Jump to live</Text>
              </Pressable>
            )}
          </View>
        </>
      )}
    </View>
  );
};
