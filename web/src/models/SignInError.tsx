const messages: Record<string, string> = {
  invitation_required: "This instance is invite-only. Ask someone on it for an invitation link.",
  invitation_invalid: "This invitation has expired or isn't valid. Ask for a new link.",
  account_disabled: "Your account is disabled. Ask whoever manages accounts here to turn it back on.",
  account_removed: "Your account was removed. Ask whoever manages accounts here to restore it.",
  access_denied: "Sign-in was cancelled.",
  state_mismatch: "That sign-in link expired. Try again.",
  sso_only: "This instance signs in through single sign-on only. Use the button for your company account.",
  sso_email_unverified: "Your company account has no verified email, which single sign-on needs. Ask whoever runs it to turn it on.",
  sso_not_ready: "Single sign-on isn't set up to create accounts yet. Ask whoever manages this instance.",
  sso_failed: "Single sign-on didn't work. Try again, or ask whoever manages this instance.",
  not_configured: "This sign-in method isn't set up here. Ask whoever manages sign-in providers to turn it on.",
};

/** Copy for the `?error=` code the server sends a failed browser sign-in back with; unknown codes get the generic line. */
export const signInErrorMessage = (code: string): string => messages[code] ?? "Sign-in didn't work. Try again.";
