import { InlineNotice } from "@/components/InlineNotice";
import { StackDeployNowButton } from "@/components/stack/StackDeployNowButton";
import type { Stack } from "@/models/Stack";

interface StackSecretsChangedNoticeProps {
  stack: Stack;
  image: string | undefined;
}

// Says which referenced secrets changed value since the last deploy; the next deploy, retry or rollback clears it.
export const StackSecretsChangedNotice = ({ stack, image }: StackSecretsChangedNoticeProps) => {
  const names = stack.secrets_changed ?? [];
  return (
    names.length > 0 && (
      <InlineNotice tone="warning" title={`${names.join(", ")} changed since the last deploy`}>
        <div className="mt-2 empty:hidden">
          <StackDeployNowButton stack={stack} image={image} label="Redeploy" />
        </div>
      </InlineNotice>
    )
  );
};
