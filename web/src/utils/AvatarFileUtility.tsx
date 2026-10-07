// Mirrors the backend's avatar caps (a person's override and a bot's avatar) so an oversized image fails before the round-trip.
const MAX_AVATAR_BYTES = 10 * 1024 * 1024;

// The picked file as the base64 data URL avatars are stored as, or the reason it can't be one.
export const readAvatarFile = (file: File): Promise<{ dataUrl: string } | { error: string }> => {
  if (!file.type.startsWith("image/")) return Promise.resolve({ error: "Please choose an image file" });
  if (file.size > MAX_AVATAR_BYTES) return Promise.resolve({ error: "Image must be under 10MB" });
  return new Promise((resolve) => {
    const reader = new FileReader();
    reader.onload = () => resolve({ dataUrl: reader.result as string });
    reader.onerror = () => resolve({ error: "That image could not be read" });
    reader.readAsDataURL(file);
  });
};
