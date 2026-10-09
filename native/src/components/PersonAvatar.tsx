import { useState } from "react";
import { Image } from "react-native";

import { personLabel, type Person } from "@nexul/client-core/person";

import { GradientTile } from "@/components/GradientTile";
import { Text } from "@/components/ui/text";
import { useApiImageSource } from "@/hooks/BotMediaHooks";
import { cn } from "@/lib/utils";

const initials = (name: string) =>
  name
    .split(/[\s._-]+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((word) => word.charAt(0).toUpperCase())
    .join("");

interface PersonAvatarProps {
  person: Person;
  // 32 in a chat stream, 20 on a ticket card, 56 on the profile.
  size: number;
  className?: string;
}

// A photo when the directory has one, else the person's seeded gradient with white initials (one initial under 24pt).
export const PersonAvatar = ({ person, size, className }: PersonAvatarProps) => {
  const served = useApiImageSource(person.avatar_url.startsWith("/api/") ? person.avatar_url : undefined);
  const remote = person.avatar_url.startsWith("https://") ? { uri: person.avatar_url } : undefined;
  const source = served ?? remote;
  const [failed, setFailed] = useState(false);
  const label = initials(personLabel(person)).slice(0, size < 24 ? 1 : 2);
  const frame = { width: size, height: size, borderRadius: size / 2 };

  return (
    <>
      {(!source || failed) && (
        <GradientTile seed={person.login} className={cn("shrink-0 rounded-full", className)}>
          <Text
            maxFontSizeMultiplier={1}
            style={{ fontSize: Math.max(10, Math.round(size * 0.38)), width: size, height: size, lineHeight: size }}
            className="text-center font-semibold text-white"
          >
            {label}
          </Text>
        </GradientTile>
      )}
      {source && !failed && (
        <Image accessibilityIgnoresInvertColors source={source} onError={() => setFailed(true)} style={frame} className={cn("shrink-0 bg-accent", className)} />
      )}
    </>
  );
};
