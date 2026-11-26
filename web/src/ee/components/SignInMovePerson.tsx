import type { MoveAccount } from "@/ee/models/SignInMove";
import { cn } from "@/lib/utils";

interface SignInMovePersonProps {
  account: MoveAccount;
  // The email to show in place of the account's own, such as the one an expected sign-in waits for.
  email?: string;
  className?: string;
}

// One account in a move: the name it goes by over the email its sign-in brings, in mono.
export const SignInMovePerson = ({ account, email, className }: SignInMovePersonProps) => (
  <span className={cn("flex min-w-0 flex-col", className)}>
    <span className="truncate text-sm font-medium text-foreground">{account.name || account.login}</span>
    <span className="truncate font-mono text-xs text-muted-foreground">{email ?? account.email}</span>
  </span>
);
