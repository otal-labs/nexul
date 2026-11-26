import { z } from "zod";

// One account as moving sign-ins shows it: the email its single sign-on brought, or the one an expected sign-in waits for.
export interface MoveAccount {
  id: string;
  login: string;
  name: string;
  email: string;
  created_at: string;
}

// Someone still on the other protocol, with the account a first sign-in through the saved one made under their email.
export interface OldSignIn extends MoveAccount {
  match: MoveAccount | null;
  expected_email: string;
}

export interface ExpectedSignIn {
  account: MoveAccount;
  created_by: string;
  created_at: string;
}

// Mirrors ee/sso's SignInMoves; protocol is empty while no provider is saved.
export interface SignInMoves {
  protocol: "oidc" | "saml" | "";
  provider_name: string;
  people: OldSignIn[];
  unmatched: MoveAccount[];
  expected: ExpectedSignIn[];
}

export interface MovePair {
  from_account_id: string;
  to_account_id: string;
}

export interface MoveResult extends MovePair {
  moved: boolean;
  message: string;
}

export const protocolName = { oidc: "OpenID Connect", saml: "SAML", "": "" } as const;

// The protocol people move from: the one not saved now.
export const otherProtocolName = (protocol: SignInMoves["protocol"]): string => {
  if (protocol === "oidc") return protocolName.saml;
  if (protocol === "saml") return protocolName.oidc;
  return "";
};

export const matchedPeople = (moves: SignInMoves): OldSignIn[] => moves.people.filter((person) => person.match);

// No suggestion yet: no account a first sign-in made matches them, and no expected sign-in waits for them.
export const unmatchedPeople = (moves: SignInMoves): OldSignIn[] => moves.people.filter((person) => !person.match && !person.expected_email);

export const ExpectSignInFormSchema = z.object({
  email: z.string().trim().pipe(z.email("Enter an email such as alice@example.com")),
});

export type ExpectSignInFormData = z.infer<typeof ExpectSignInFormSchema>;
