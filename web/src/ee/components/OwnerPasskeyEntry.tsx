import { Link, useSearchParams } from "react-router";

import { PasskeySignInButton } from "@/components/auth/PasskeySignInButton";

// While single sign-on is the only sign-in, the instance Owner's passkey is a link under it, so the page lists no other method.
export const OwnerPasskeyEntry = () => {
  const [searchParams] = useSearchParams();
  const asOwner = searchParams.has("owner");
  return (
    <>
      {asOwner && <PasskeySignInButton />}
      {!asOwner && (
        <p className="text-row text-muted-foreground">
          <Link to="/login?owner" className="underline underline-offset-4 hover:text-foreground">
            Instance Owner? Use your passkey
          </Link>
        </p>
      )}
    </>
  );
};
