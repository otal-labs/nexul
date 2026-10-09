import { Tabs } from "expo-router";
import Ellipsis from "lucide-react-native/icons/ellipsis";
import Inbox from "lucide-react-native/icons/inbox";
import MessageSquare from "lucide-react-native/icons/message-square";
import Rocket from "lucide-react-native/icons/rocket";
import SquareKanban from "lucide-react-native/icons/square-kanban";
import { useCSSVariable } from "uniwind";

import type { Area } from "@nexul/client-core/permissions";

import { OFFLINE_BANNER_HEIGHT, useIsOffline } from "@/components/OfflineBanner";
import { TabBar } from "@/components/TabBar";
import { unreadBadge, useFetchUnreadCount } from "@/hooks/NotificationHooks";
import { useAreaAccess, useEnsureWorkspaceSelected } from "@/hooks/WorkspaceHooks";

export default function TabsLayout() {
  const [brand, mutedForeground, panel, border, foreground, background] = useCSSVariable([
    "--color-brand",
    "--color-muted-foreground",
    "--color-panel",
    "--color-border",
    "--color-foreground",
    "--color-background",
  ]);
  const offline = useIsOffline();
  useEnsureWorkspaceSelected();
  const { data: unread } = useFetchUnreadCount();
  const badge = unreadBadge(unread?.count);
  const can = useAreaAccess();
  // href null drops a tab from the bar; it stays hidden until the viewer's permissions say it may be read.
  const hiddenUnless = (area: Area) => (can?.(area) ? {} : { href: null });

  return (
    <Tabs
      tabBar={(props) => <TabBar {...props} />}
      screenOptions={{
        headerShown: false,
        sceneStyle: { paddingBottom: offline ? OFFLINE_BANNER_HEIGHT : 0 },
        // The active tab is the ember's "where you are"; the unread count is the solid ink pill.
        tabBarActiveTintColor: String(brand),
        tabBarInactiveTintColor: String(mutedForeground),
        tabBarLabelStyle: { fontFamily: "Inter", fontWeight: "500", fontSize: 11 },
        tabBarStyle: { backgroundColor: String(panel), borderTopColor: String(border), elevation: 0 },
        tabBarBadgeStyle: { backgroundColor: String(foreground), color: String(background), fontFamily: "JetBrains Mono", fontSize: 10 },
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
          ...hiddenUnless("tickets"),
        }}
      />
      <Tabs.Screen
        name="deploys"
        options={{
          title: "Deploys",
          tabBarIcon: ({ color, size }) => <Rocket color={color} size={size} />,
          ...hiddenUnless("stacks"),
        }}
      />
      <Tabs.Screen
        name="more"
        options={{ title: "More", tabBarIcon: ({ color, size }) => <Ellipsis color={color} size={size} /> }}
      />
    </Tabs>
  );
}
