import { useRouter } from "expo-router";

import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";

export const SignOutButton = () => {
  const router = useRouter();

  return (
    <Button variant="destructive" onPress={() => router.push({ pathname: "/more/settings/sign-out", params: { mode: "current" } })}>
      <Text>Sign out</Text>
    </Button>
  );
};
