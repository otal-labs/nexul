import { z } from "zod";

// What the Owner reads back; the client secret is never sent, only whether one is stored.
export const SsoSettingsSchema = z.object({
  unlocked: z.boolean(),
  configured: z.boolean(),
  issuer_url: z.string(),
  client_id: z.string(),
  client_secret_set: z.boolean(),
  display_name: z.string(),
  join_role_id: z.string(),
  only: z.boolean(),
  redirect_uri: z.string(),
  active: z.boolean(),
});

export type SsoSettings = z.infer<typeof SsoSettingsSchema>;

// What the sign-in page reads before anyone is signed in.
export const SsoPublicSchema = z.object({ active: z.boolean(), name: z.string(), only: z.boolean() });

export type SsoPublic = z.infer<typeof SsoPublicSchema>;

export const SsoMaxNameLength = 40;

// A saved provider keeps its secret when the field is left blank.
export const ssoFormSchema = (hasSecret: boolean) =>
  z.object({
    issuer_url: z.string().trim().min(1, "Enter the issuer URL"),
    client_id: z.string().trim().min(1, "Enter the client ID"),
    client_secret: hasSecret ? z.string().trim() : z.string().trim().min(1, "Enter the client secret"),
    display_name: z.string().trim().min(1, "Enter the name the button shows").max(SsoMaxNameLength, `At most ${SsoMaxNameLength} characters`),
    join_role_id: z.string().min(1, "Choose the role new people get"),
  });

export type SsoFormData = z.infer<ReturnType<typeof ssoFormSchema>>;

export const ssoChecks = [
  { key: "issuer", label: "The provider answers at this issuer URL", why: "Its sign-in endpoints are read from the address." },
  { key: "secret", label: "The client secret reads", why: "A secret reference is read from its manager now." },
];
