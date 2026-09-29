import { Tabs } from "expo-router";
import { Ellipsis, Inbox, MessageSquare, Rocket, SquareKanban } from "lucide-react-native";
import { useCSSVariable } from "uniwind";

import { OFFLINE_BANNER_HEIGHT, useIsOffline } from "@/components/OfflineBanner";
import { TabBar } from "@/components/TabBar";
import { unreadBadge, useFetchUnreadCount } from "@/hooks/NotificationHooks";
import { useEnsureWorkspaceSelected } from "@/hooks/WorkspaceHooks";

export default function TabsLayout() {
  const [foreground, mutedForeground] = useCSSVariable([
    "--color-foreground",
    "--color-muted-foreground",
  ]);
  const offline = useIsOffline();
  useEnsureWorkspaceSelected();
  const { data: unread } = useFetchUnreadCount();
  const badge = unreadBadge(unread?.count);

  return (
    <Tabs
      tabBar={(props) => <TabBar {...props} />}
      screenOptions={{
        headerShown: false,
        sceneStyle: { paddingBottom: offline ? OFFLINE_BANNER_HEIGHT : 0 },
        tabBarActiveTintColor: String(foreground),
        tabBarInactiveTintColor: String(mutedForeground),
        tabBarLabelStyle: { fontFamily: "Inter", fontWeight: "500" },
      }}
    >
      <Tabs.Screen
        name="inbox"
        options={{
          title: "Inbox",
          tabBarIcon: ({ color, size }) => <Inbox color={color} size={size} />,
          // exactOptionalPropertyTypes rejects an explicit undefined, so the key is only present with a badge.
          ...(badge !== undefined && { tabBarBadge: badge }),
        }}
      />
      <Tabs.Screen
        name="chat"
        options={{
          title: "Chat",
          tabBarIcon: ({ color, size }) => <MessageSquare color={color} size={size} />,
        }}
      />
      <Tabs.Screen
        name="board"
        options={{
          title: "Board",
          tabBarIcon: ({ color, size }) => <SquareKanban color={color} size={size} />,
        }}
      />
      <Tabs.Screen
        name="deploys"
        options={{ title: "Deploys", tabBarIcon: ({ color, size }) => <Rocket color={color} size={size} /> }}
      />
      <Tabs.Screen
        name="more"
        options={{ title: "More", tabBarIcon: ({ color, size }) => <Ellipsis color={color} size={size} /> }}
      />
    </Tabs>
  );
}
