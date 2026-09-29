import { useLocalSearchParams, useRouter } from "expo-router";
import { View } from "react-native";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { SheetTitle } from "@/components/SheetTitle";
import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import { useSignOut, useSignOutOtherSessions, useSignOutSession } from "@/hooks/SessionHooks";
import { useSessionStore } from "@/stores/sessionStore";

type SignOutParams = { mode: "device" | "others" | "current"; id?: string; label?: string };

export const SignOutSheet = () => {
  const router = useRouter();
  const { mode, id, label } = useLocalSearchParams<SignOutParams>();
  const host = useSessionStore((s) => s.host);
  const signOutDevice = useSignOutSession();
  const signOutOthers = useSignOutOtherSessions();
  const signOutCurrent = useSignOut();

  const title = {
    device: `Sign out ${label}?`,
    others: "Sign out everywhere else?",
    current: `Sign out of ${host?.replace(/^https?:\/\//, "") ?? "Nexul"}?`,
  }[mode];
  const body = {
    device: "That device will need to sign in again.",
    others: "Every other device will need to sign in again.",
    current: "You'll need to scan the code again from Devices to reconnect.",
  }[mode];
  const action = { device: signOutDevice, others: signOutOthers, current: signOutCurrent }[mode];
  const run = () => {
    if (mode === "device" && id) return signOutDevice.mutate(id, { onSuccess: () => router.back() });
    if (mode === "others") return signOutOthers.mutate(undefined, { onSuccess: () => router.back() });
    signOutCurrent.mutate();
  };

  return (
    <View className="gap-4 bg-popover px-4 pb-6">
      <SheetTitle title={title} className="px-0 pb-0" />
      <Text variant="muted">{body}</Text>
      {action.error && <ErrorDisplay error={action.error} />}
      <Button variant="destructive" disabled={action.isPending} onPress={run}>
        <Text>{action.isPending ? "Signing out…" : "Sign out"}</Text>
      </Button>
      <Button variant="outline" disabled={action.isPending} onPress={() => router.back()}>
        <Text>Cancel</Text>
      </Button>
    </View>
  );
};
