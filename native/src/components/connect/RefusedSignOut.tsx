import { useState } from "react";
import { View } from "react-native";

import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import { useSignOut } from "@/hooks/SessionHooks";

// The refused screen sits above the navigator, so there is no sheet route to push: the confirm opens in place.
export const RefusedSignOut = () => {
  const [confirming, setConfirming] = useState(false);
  const signOut = useSignOut();

  return (
    <View className="gap-3">
      {!confirming && (
        <Button variant="destructive" onPress={() => setConfirming(true)}>
          <Text>Sign out</Text>
        </Button>
      )}
      {confirming && (
        <>
          <Text variant="muted">You will need to scan the code again from Devices to reconnect.</Text>
          <Button variant="destructive" disabled={signOut.isPending} onPress={() => signOut.mutate()}>
            <Text>{signOut.isPending ? "Signing out…" : "Confirm sign out"}</Text>
          </Button>
          <Button variant="outline" disabled={signOut.isPending} onPress={() => setConfirming(false)}>
            <Text>Cancel</Text>
          </Button>
        </>
      )}
    </View>
  );
};
