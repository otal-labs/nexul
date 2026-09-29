const messages: Record<string, string> = {
  invitation_required: "This instance is invite-only. Ask someone on it for an invitation link, then open that link to join.",
  invitation_invalid: "This invitation is invalid or has expired.",
  account_disabled: "Your account on this instance is disabled. Ask whoever manages accounts on it to reactivate it.",
  account_removed: "Your account was removed from this instance. Ask whoever manages accounts on it to restore it.",
  access_denied: "Sign-in was cancelled.",
  state_mismatch: "That sign-in link expired. Try again.",
  not_configured: "This sign-in method isn't set up on this instance. Ask the instance's Owner to enable it.",
};

/** Copy for the `?error=` code the server sends a failed browser sign-in back with; unknown codes get the generic line. */
export const signInErrorMessage = (code: string): string => messages[code] ?? "Sign-in didn't work. Try again.";
