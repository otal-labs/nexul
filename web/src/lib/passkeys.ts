// The browser's own JSON helpers (2024 and later); a browser without them shows no passkey controls.
export const passkeysSupported = (): boolean =>
  typeof window.PublicKeyCredential?.parseCreationOptionsFromJSON === "function" &&
  typeof window.PublicKeyCredential.parseRequestOptionsFromJSON === "function";

const asPublicKey = (credential: Credential | null): PublicKeyCredential => {
  if (!(credential instanceof PublicKeyCredential)) throw new Error("The browser returned no passkey");
  return credential;
};

export const createPasskey = async (options: PublicKeyCredentialCreationOptionsJSON) =>
  asPublicKey(await navigator.credentials.create({ publicKey: PublicKeyCredential.parseCreationOptionsFromJSON(options) })).toJSON();

export const getPasskey = async (options: PublicKeyCredentialRequestOptionsJSON) =>
  asPublicKey(await navigator.credentials.get({ publicKey: PublicKeyCredential.parseRequestOptionsFromJSON(options) })).toJSON();

// Closing the browser's prompt, or letting it time out, is the person's choice, not an error to report.
export const passkeyCancelled = (error: unknown): boolean => error instanceof DOMException && error.name === "NotAllowedError";
