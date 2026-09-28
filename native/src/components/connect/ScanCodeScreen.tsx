import { CameraView, useCameraPermissions, type BarcodeScanningResult } from "expo-camera";
import { useRouter } from "expo-router";
import { useRef, useState } from "react";
import { View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";

import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import { parseConnectLink } from "@/lib/connectLink";

export const ScanCodeScreen = () => {
  const [permission, requestPermission] = useCameraPermissions();
  const router = useRouter();
  const insets = useSafeAreaInsets();
  const handled = useRef(false);
  const [rejected, setRejected] = useState(false);

  const onBarcodeScanned = ({ data }: BarcodeScanningResult) => {
    if (handled.current) return;
    const link = parseConnectLink(data);
    if (!link) {
      setRejected(true);
      return;
    }
    // The camera keeps reporting the same code every frame; only the first hit navigates.
    handled.current = true;
    router.replace({ pathname: "/connect/manual", params: { host: link.host, code: link.code, auto: "1" } });
  };

  return (
    <View className="flex-1 bg-background">
      {!permission && <LoadingDisplay />}
      {permission && !permission.granted && (
        <View className="flex-1 justify-center gap-4 px-6">
          <Text variant="muted" className="leading-5">
            The camera reads the code shown on the Devices page. Nothing is recorded.
          </Text>
          <Button variant="outline" onPress={() => void requestPermission()}>
            <Text>Allow the camera</Text>
          </Button>
        </View>
      )}
      {permission?.granted && (
        <CameraView
          style={{ flex: 1 }}
          facing="back"
          barcodeScannerSettings={{ barcodeTypes: ["qr"] }}
          onBarcodeScanned={onBarcodeScanned}
        />
      )}
      {permission?.granted && (
        <View className="border-t border-border bg-background px-6 pt-4" style={{ paddingBottom: insets.bottom + 16 }}>
          {!rejected && <Text variant="muted">Point the camera at the QR code.</Text>}
          {rejected && <Text variant="muted">That code is not a Nexul connect code.</Text>}
        </View>
      )}
    </View>
  );
};
