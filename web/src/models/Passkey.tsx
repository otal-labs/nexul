import { z } from "zod";

export interface Passkey {
  id: string;
  name: string;
  // The domain it was made for; it signs in nowhere else.
  rp_id: string;
  created_at: string;
  last_used_at?: string;
}

export interface PasskeysResponse {
  passkeys: Passkey[];
  // Whether this address can make passkeys at all: https, or localhost.
  available: boolean;
}

export interface PasskeyCeremony<TOptions> {
  ceremony: string;
  options: TOptions;
}

export const RenamePasskeyFormSchema = z.object({
  name: z.string().trim().min(1, "Name is required").max(64, "64 characters at most"),
});
export type RenamePasskeyFormData = z.infer<typeof RenamePasskeyFormSchema>;
