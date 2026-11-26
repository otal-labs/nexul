import { Trash2 } from "lucide-react";

import { ConfirmDestroyButton } from "@/components/settings/ConfirmDestroyButton";
import { SignInMovePerson } from "@/ee/components/SignInMovePerson";
import { useRemoveExpectedSignIn } from "@/ee/hooks/SignInMoveHooks";
import type { ExpectedSignIn } from "@/ee/models/SignInMove";
import { formatShortDate } from "@/utils/TimeUtility";

// One expected sign-in waiting: whose account, the email it waits for, since when, and the way to take it back.
export const ExpectedSignInRow = ({ expected }: { expected: ExpectedSignIn }) => {
  const remove = useRemoveExpectedSignIn();
  return (
    <li className="flex items-center gap-3 py-3">
      <SignInMovePerson account={expected.account} className="flex-1" />
      <span className="shrink-0 font-mono text-xs text-muted-foreground tabular-nums">since {formatShortDate(expected.created_at)}</span>
      <ConfirmDestroyButton
        icon={Trash2}
        idleLabel="Remove expected sign-in"
        confirmLabel="Remove"
        loading={remove.isPending}
        onConfirm={() => remove.mutate(expected.account.id)}
        gate={{ permission: "sign_ins:move", verb: "remove an expected sign-in" }}
      />
    </li>
  );
};
