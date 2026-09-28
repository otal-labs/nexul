import { Tabs } from "expo-router";
import { Ellipsis, Inbox, MessageSquare, Rocket, SquareKanban } from "lucide-react-native";
import { useCSSVariable } from "uniwind";

export default function TabsLayout() {
  const [foreground, mutedForeground] = useCSSVariable([
    "--color-foreground",
    "--color-muted-foreground",
  ]);

  return (
    <Tabs
      screenOptions={{
        headerShown: false,
        tabBarActiveTintColor: String(foreground),
        tabBarInactiveTintColor: String(mutedForeground),
        tabBarLabelStyle: { fontFamily: "Inter", fontWeight: "500" },
      }}
    >
      <Tabs.Screen
        name="inbox"
        options={{ title: "Inbox", tabBarIcon: ({ color, size }) => <Inbox color={color} size={size} /> }}
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
