import { Redirect, useLocalSearchParams, useRouter } from "expo-router";
import QrCode from "lucide-react-native/icons/qr-code";
import { View } from "react-native";
import { useCSSVariable } from "uniwind";

import { FieldScreen } from "@/components/FieldScreen";
import { NexulMark } from "@/components/NexulMark";
import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";

type ConnectParams = {
  host?: string;
  code?: string;
};

// The phone's showcase surface: the light field, the mark, a display headline, one primary action.
export const ConnectScreen = () => {
  const { host, code } = useLocalSearchParams<ConnectParams>();
  const router = useRouter();
  const [foreground, brandForeground] = useCSSVariable(["--color-foreground", "--color-brand-foreground"]);

  return (
    <FieldScreen>
      {host && code && <Redirect href={{ pathname: "/connect/manual", params: { host, code, auto: "1" } }} />}
      <View className="flex-1 justify-end gap-10 px-6 pb-10">
        <View className="gap-4">
          <NexulMark size={40} color={String(foreground)} />
          <Text role="heading" className="font-display text-[38px] leading-[42px] tracking-[-0.8px]">
            Connect this phone to Nexul
          </Text>
          <Text className="text-base leading-6 text-muted-foreground">
            On the web, open Your settings, Security, Devices and show the connect code. This phone signs in as you.
          </Text>
        </View>
        <View className="gap-3">
          <Button onPress={() => router.push("/connect/scan")}>
            <QrCode size={18} color={String(brandForeground)} />
            <Text>Scan the code</Text>
          </Button>
          <Button variant="ghost" onPress={() => router.push("/connect/manual")}>
            <Text>Enter it by hand</Text>
          </Button>
        </View>
      </View>
    </FieldScreen>
  );
};
