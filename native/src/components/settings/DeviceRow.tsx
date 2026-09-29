import { Monitor, Smartphone, X } from "lucide-react-native";
import { Pressable, View } from "react-native";
import { useCSSVariable } from "uniwind";

import { Text } from "@/components/ui/text";
import { RelativeTime } from "@/components/RelativeTime";
import type { Session } from "@/models/User";

const PHONE_PLATFORMS = ["Android", "iOS"];

export const deviceLabel = (session: Session) => [session.platform, session.label].filter(Boolean).join(" · ");

interface DeviceRowProps {
  session: Session;
  onSignOut?: () => void;
}

export const DeviceRow = ({ session, onSignOut }: DeviceRowProps) => {
  const [mutedForeground] = useCSSVariable(["--color-muted-foreground"]);
  const phone = session.client === "phone" || PHONE_PLATFORMS.includes(session.platform);
  const label = deviceLabel(session);

  return (
    <View className="min-h-11 flex-row items-center gap-3 border-b border-border bg-card px-4 py-3">
      {phone && <Smartphone size={16} color={String(mutedForeground)} />}
      {!phone && <Monitor size={16} color={String(mutedForeground)} />}
      <View className="min-w-0 flex-1">
        <View className="flex-row items-center gap-2">
          <Text numberOfLines={1} className="flex-1 font-medium">
            {label}
          </Text>
          {session.current && (
            <Text variant="muted" className="font-mono text-[10px] uppercase tracking-wide">
              This device
            </Text>
          )}
        </View>
        <Text variant="muted" numberOfLines={1} className="font-mono text-xs">
          {session.ip} · {session.current ? "active now" : <RelativeTime iso={session.last_active_at} />}
        </Text>
      </View>
      {!session.current && onSignOut && (
        <Pressable role="button" accessibilityLabel="Sign out" onPress={onSignOut} className="size-11 items-center justify-center">
          <X size={16} color={String(mutedForeground)} />
        </Pressable>
      )}
    </View>
  );
};
