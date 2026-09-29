import { View } from "react-native";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Text } from "@/components/ui/text";

interface ManualCodeFormProps {
  host: string;
  code: string;
  pending: boolean;
  error: unknown;
  onHostChange: (host: string) => void;
  onCodeChange: (code: string) => void;
  onSubmit: () => void;
}

export const ManualCodeForm = ({ host, code, pending, error, onHostChange, onCodeChange, onSubmit }: ManualCodeFormProps) => {
  const canSubmit = host.trim() !== "" && code.trim() !== "" && !pending;
  return (
    <View className="gap-5 px-6 pt-6">
      <View className="gap-2">
        <Text variant="small" className="text-muted-foreground">
          Instance address
        </Text>
        <Input
          value={host}
          onChangeText={onHostChange}
          placeholder="https://nexul.example.com"
          autoCapitalize="none"
          autoCorrect={false}
          keyboardType="url"
          textContentType="URL"
          editable={!pending}
          className="font-mono"
          aria-label="Instance address"
        />
      </View>
      <View className="gap-2">
        <Text variant="small" className="text-muted-foreground">
          Connect code
        </Text>
        <Input
          value={code}
          onChangeText={onCodeChange}
          placeholder="XXXX-XXXX-XXXX"
          autoCapitalize="characters"
          autoCorrect={false}
          editable={!pending}
          className="font-mono"
          aria-label="Connect code"
        />
      </View>
      {error !== null && error !== undefined && <ErrorDisplay error={error} className="px-0" />}
      <Button onPress={onSubmit} disabled={!canSubmit}>
        <Text>{pending ? "Connecting…" : "Connect"}</Text>
      </Button>
    </View>
  );
};
