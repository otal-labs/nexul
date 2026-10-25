import { z } from "zod";

// Mirrors internal/harness SecretRequest: the agent asking for a value that goes straight to its harness.
export interface HarnessSecretRequest {
  id: string;
  label: string;
  reason: string;
  placeholder?: string;
}

export const SaveSecretFormSchema = z.object({ secret: z.string().trim().min(1, "Enter the value") });

export type SaveSecretFormData = z.infer<typeof SaveSecretFormSchema>;

const SECRET_FENCE = "```nexul-secret\n";

// The Agent posts a secret request as its own message with the request JSON fenced; anything else is a plain message.
export const parseSecretMessage = (body: string): HarnessSecretRequest | null => {
  if (!body.startsWith(SECRET_FENCE) || !body.trimEnd().endsWith("```")) return null;
  const json = body.slice(SECRET_FENCE.length, body.trimEnd().length - 3);
  try {
    const parsed: unknown = JSON.parse(json);
    if (typeof parsed !== "object" || parsed === null || !("id" in parsed) || !("label" in parsed)) return null;
    return parsed as HarnessSecretRequest;
  } catch {
    return null;
  }
};
