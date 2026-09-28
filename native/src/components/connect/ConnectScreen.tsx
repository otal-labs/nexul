import { Redirect, useLocalSearchParams, useRouter } from "expo-router";
import { View } from "react-native";

import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";

type ConnectParams = {
  host?: string;
  code?: string;
};

export const ConnectScreen = () => {
  const { host, code } = useLocalSearchParams<ConnectParams>();
  const router = useRouter();

  return (
    <View className="flex-1 justify-center gap-6 bg-background px-6">
      {host && code && <Redirect href={{ pathname: "/connect/manual", params: { host, code, auto: "1" } }} />}
      <View className="gap-2">
        <Text variant="h4">Connect this phone</Text>
        <Text variant="muted" className="leading-5">
          Scan the code in Nexul → Settings → Security → Devices.
        </Text>
      </View>
      <View className="gap-3">
        <Button onPress={() => router.push("/connect/scan")}>
          <Text>Scan the code</Text>
        </Button>
        <Button variant="outline" onPress={() => router.push("/connect/manual")}>
          <Text>Enter it by hand</Text>
        </Button>
      </View>
    </View>
  );
};
