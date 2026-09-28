import { useLocalSearchParams } from "expo-router";
import { useEffect, useRef, useState } from "react";
import { View } from "react-native";

import { ManualCodeForm } from "@/components/connect/ManualCodeForm";
import { ServerRefusedScreen } from "@/components/connect/ServerRefusedScreen";
import { useConnectPhone } from "@/hooks/ConnectHooks";
import { normalizeHost } from "@/lib/connectLink";

type ManualCodeParams = {
  host?: string;
  code?: string;
  auto?: string;
};

export const ManualCodeScreen = () => {
  const params = useLocalSearchParams<ManualCodeParams>();
  const [host, setHost] = useState(params.host ?? "");
  const [code, setCode] = useState(params.code ?? "");
  const { mutate, data, error, isPending } = useConnectPhone();
  const submit = () => mutate({ host: normalizeHost(host), code: code.trim() });
  const autoSubmitted = useRef(false);

  // A scanned or deep-linked code is submitted once without a second tap; a repeat would count as a wrong code.
  useEffect(() => {
    if (autoSubmitted.current || params.auto !== "1" || !params.host || !params.code) return;
    autoSubmitted.current = true;
    mutate({ host: normalizeHost(params.host), code: params.code.trim() });
  }, [params.auto, params.host, params.code, mutate]);

  const refused = data?.kind === "refused" ? data : undefined;

  return (
    <View className="flex-1 bg-background">
      {refused && (
        <ServerRefusedScreen host={refused.host} version={refused.version} retrying={isPending} onRetry={submit} />
      )}
      {!refused && (
        <ManualCodeForm
          host={host}
          code={code}
          pending={isPending}
          error={error}
          onHostChange={setHost}
          onCodeChange={setCode}
          onSubmit={submit}
        />
      )}
    </View>
  );
};
