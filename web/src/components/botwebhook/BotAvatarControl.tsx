import { useRef, useState, type ChangeEvent } from "react";

import { BotAvatar } from "@/components/botwebhook/BotAvatar";
import { Button } from "@/components/ui/button";
import { readAvatarFile } from "@/utils/AvatarFileUtility";

interface BotAvatarControlProps {
  avatar: string;
  /** A data URL, or "" for the default glyph. */
  onChange: (avatar: string) => void;
}

export const BotAvatarControl = ({ avatar, onChange }: BotAvatarControlProps) => {
  const input = useRef<HTMLInputElement>(null);
  const [error, setError] = useState<string | null>(null);

  const onFile = async (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.currentTarget.files?.[0];
    e.currentTarget.value = "";
    if (!file) return;
    const read = await readAvatarFile(file);
    if ("error" in read) {
      setError(read.error);
      return;
    }
    setError(null);
    onChange(read.dataUrl);
  };

  return (
    <div className="space-y-1.5">
      <div className="flex items-center gap-3">
        <BotAvatar avatar={avatar} className="size-12" />
        <div className="flex flex-wrap gap-1">
          <Button type="button" variant="outline" size="sm" onClick={() => input.current?.click()}>
            {avatar ? "Change avatar" : "Pick an avatar"}
          </Button>
          {avatar && (
            <Button type="button" variant="ghost" size="sm" onClick={() => onChange("")}>
              Use the default
            </Button>
          )}
        </div>
        <input ref={input} type="file" accept="image/*" aria-label="Avatar image" hidden onChange={(e) => void onFile(e)} />
      </div>
      {error && (
        <p role="alert" className="text-xs text-destructive">
          {error}
        </p>
      )}
    </div>
  );
};
