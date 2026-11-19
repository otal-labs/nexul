import { useEffect } from "react";
import { useLocation, useNavigate } from "react-router";

import { SignedOutFrame } from "@/components/auth/SignedOutFrame";
import { OAuthConsentSection } from "@/components/oauth/OAuthConsentSection";
import { useSessionStore } from "@/stores/sessionStore";
import { useSignInReturnStore } from "@/stores/signInReturnStore";

// An MCP client sends the browser here to ask for a grant; signed out, the person signs in first and comes back.
export const OAuthAuthorizePage = () => {
  const { pathname, search } = useLocation();
  const navigate = useNavigate();
  const isLoggedIn = useSessionStore((s) => s.isLoggedIn);
  const keep = useSignInReturnStore((s) => s.keep);

  useEffect(() => {
    if (isLoggedIn) return;
    keep(`${pathname}${search}`);
    navigate("/login", { replace: true });
  }, [isLoggedIn, keep, navigate, pathname, search]);

  return <SignedOutFrame card>{isLoggedIn && <OAuthConsentSection search={search} />}</SignedOutFrame>;
};
