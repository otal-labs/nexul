import { View } from "react-native";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PersonAvatar } from "@/components/PersonAvatar";
import { SettingsCard } from "@/components/SettingsCard";
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
  const user = me.data?.user;
  const name = user ? user.display_name || user.name || user.login : "";

  return (
    <View className="gap-4">
      {isPending && <LoadingDisplay message="Loading your profile" />}
      {error && <ErrorDisplay error={error} className="px-0" />}
      {user && (
        <View className="flex-row items-center gap-3.5">
          <PersonAvatar person={{ user_id: user.id, login: user.login, display_name: name, avatar_url: effectiveAvatar(user) }} size={56} />
          <View className="min-w-0 flex-1 gap-0.5">
            <Text numberOfLines={1} className="text-lg font-semibold">
              {name}
            </Text>
            <Text numberOfLines={1} className="font-mono text-xs text-muted-foreground">
              {user.login}
            </Text>
          </View>
        </View>
      )}
      {identities.data && identities.data.identities.length > 0 && (
        <SettingsCard title="Sign-in accounts">
          {identities.data.identities.map((identity, i) => (
            <SignInAccountRow key={identity.provider} identity={identity} first={i === 0} />
          ))}
        </SettingsCard>
      )}
    </View>
  );
};
