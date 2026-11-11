import { secretManagerName } from "@nexul/client-core/secretReference";

import { InlineNotice } from "@/components/InlineNotice";
import { useRunners } from "@/hooks/RunnerHooks";
import type { Stack } from "@/models/Stack";
import { unreadableManagers } from "@/utils/SecretReaderUtility";

interface StackSecretReaderWarningProps {
  stack: Stack;
}

// Says a stack that holds secret references is on a machine none of whose runners can read them, which is what refuses its deploys.
export const StackSecretReaderWarning = ({ stack }: StackSecretReaderWarningProps) => {
  const { data: runners } = useRunners();
  const missing = runners ? unreadableManagers(stack.env, stack.machine, runners) : [];
  return (
    missing.length > 0 && (
      <InlineNotice tone="warning" title={`No runner on ${stack.machine} can read ${missing.map(secretManagerName).join(" or ")} references yet`}>
        Deploys will be refused until one has a token.
      </InlineNotice>
    )
  );
};
