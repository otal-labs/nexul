import { Image, View } from "react-native";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { SignInAccountRow } from "@/components/settings/SignInAccountRow";
import { Text } from "@/components/ui/text";
import { useFetchIdentities, useFetchMe } from "@/hooks/AuthHooks";
import { effectiveAvatar } from "@/models/User";

const initials = (name: string) =>
  name
    .split(/\s+/)
    .slice(0, 2)
    .map((word) => word.charAt(0))
    .join("")
    .toUpperCase();

// Read-only, per ticket 31: no edit form here, linking/unlinking a provider stays web-only for now.
export const ProfileSection = () => {
  const me = useFetchMe(true);
  const identities = useFetchIdentities();
  const isPending = me.isPending || identities.isPending;
  const error = me.error ?? identities.error;
  const avatar = me.data && effectiveAvatar(me.data.user);
  const name = me.data ? me.data.user.display_name || me.data.user.name || me.data.user.login : "";

  return (
    <View className="gap-4 px-4 py-4">
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} className="px-0" />}
      {me.data && (
        <View className="flex-row items-center gap-3">
          {avatar && <Image source={{ uri: avatar }} className="size-14 rounded-full bg-muted" />}
          {!avatar && (
            <View className="size-14 items-center justify-center rounded-full bg-muted">
              <Text variant="large" className="font-mono">
                {initials(name)}
              </Text>
            </View>
          )}
          <Text variant="large">{name}</Text>
        </View>
      )}
      {identities.data && identities.data.identities.length > 0 && (
        <View className="overflow-hidden rounded-md border border-border">
          {identities.data.identities.map((identity) => (
            <SignInAccountRow key={identity.provider} identity={identity} />
          ))}
        </View>
      )}
    </View>
  );
};
