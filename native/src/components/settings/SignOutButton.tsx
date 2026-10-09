import { useRouter } from "expo-router";

import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";

// An outline in the destructive hue: the solid red belongs to the confirm sheet, where signing out is the one thing left to do.
export const SignOutButton = () => {
  const router = useRouter();

  return (
    <Button variant="outline" className="border-destructive/40" onPress={() => router.push({ pathname: "/more/settings/sign-out", params: { mode: "current" } })}>
      <Text className="text-destructive">Sign out</Text>
    </Button>
  );
};
