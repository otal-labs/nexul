import { useRouter } from "expo-router";
import CircleHelp from "lucide-react-native/icons/circle-question-mark";

import { EmptyState } from "@/components/EmptyState";
import { FieldScreen } from "@/components/FieldScreen";
import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";

// A link the app has no screen for: the empty state, and the way back to the Inbox.
export const NotFoundScreen = () => {
  const router = useRouter();
  return (
    <FieldScreen>
      <EmptyState
        icon={CircleHelp}
        title="This page doesn't exist"
        message="The link may be from a newer version of Nexul, or the page moved."
        action={
          <Button onPress={() => router.replace("/inbox")}>
            <Text>Go to Inbox</Text>
          </Button>
        }
      />
    </FieldScreen>
  );
};
