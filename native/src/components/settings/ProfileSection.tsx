import { Image, View } from "react-native";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { SignInAccountRow } from "@/components/settings/SignInAccountRow";
import { Text } from "@/components/ui/text";
import { useFetchIdentities, useFetchMe } from "@/hooks/AuthHooks";
import { effectiveAvatar } from "@/models/User";

// Read-only, per ticket 31: no edit form here, linking/unlinking a provider stays web-only for now.
export const ProfileSection = () => {
  const me = useFetchMe(true);
  const identities = useFetchIdentities();
  const isPending = me.isPending || identities.isPending;
  const error = me.error ?? identities.error;

  return (
    <View className="gap-4 px-4 py-4">
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {me.data && (
        <View className="flex-row items-center gap-3">
          <Image source={{ uri: effectiveAvatar(me.data.user) }} className="size-14 rounded-full bg-muted" />
          <Text variant="large">{me.data.user.display_name || me.data.user.name}</Text>
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
