import { Monitor, Smartphone, X } from "lucide-react-native";
import { Pressable, View } from "react-native";
import { useCSSVariable } from "uniwind";

import { RelativeTime } from "@/components/RelativeTime";
import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";
import type { Session } from "@/models/User";

const PHONE_PLATFORMS = ["Android", "iOS"];

export const deviceLabel = (session: Session) => [session.platform, session.label].filter(Boolean).join(" · ");

interface DeviceRowProps {
  session: Session;
  first: boolean;
  onSignOut?: () => void;
}

export const DeviceRow = ({ session, first, onSignOut }: DeviceRowProps) => {
  const [mutedForeground] = useCSSVariable(["--color-muted-foreground"]);
  const phone = session.client === "phone" || PHONE_PLATFORMS.includes(session.platform);
  const label = deviceLabel(session);

  return (
    <View className={cn("min-h-16 flex-row items-center gap-3 py-2.5 pl-4 pr-1", !first && "border-t border-border")}>
      {phone && <Smartphone size={18} color={String(mutedForeground)} />}
      {!phone && <Monitor size={18} color={String(mutedForeground)} />}
      <View className="min-w-0 flex-1 gap-0.5">
        <Text numberOfLines={1} className="text-[15px]">
          {label}
        </Text>
        <Text numberOfLines={1} className="font-mono text-xs text-muted-foreground">
          {session.ip} · {session.current ? "active now" : <RelativeTime iso={session.last_active_at} />}
        </Text>
      </View>
      {session.current && <Text className="pr-3 font-mono text-[11px] uppercase tracking-[1px] text-muted-foreground">This phone</Text>}
      {!session.current && onSignOut && (
        <Pressable role="button" accessibilityLabel={`Sign out ${label}`} onPress={onSignOut} className="size-11 items-center justify-center rounded-md active:bg-accent">
          <X size={16} color={String(mutedForeground)} />
        </Pressable>
      )}
    </View>
  );
};
