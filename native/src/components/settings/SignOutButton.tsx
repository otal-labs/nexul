import { Alert } from "react-native";

import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import { useSignOut } from "@/hooks/SessionHooks";

export const SignOutButton = () => {
  const signOut = useSignOut();

  const confirm = () =>
    Alert.alert("Sign out?", "You'll need to scan the code again from Devices to reconnect.", [
      { text: "Cancel", style: "cancel" },
      { text: "Sign out", style: "destructive", onPress: () => signOut.mutate() },
    ]);

  return (
    <Button variant="destructive" disabled={signOut.isPending} onPress={confirm}>
      <Text>{signOut.isPending ? "Signing out…" : "Sign out"}</Text>
    </Button>
  );
};
