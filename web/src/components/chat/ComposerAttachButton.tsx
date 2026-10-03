import { ImagePlus } from "lucide-react";
import { useRef, type ChangeEvent } from "react";

import { Button } from "@/components/ui/button";

export const ComposerAttachButton = ({ onFiles }: { onFiles: (files: File[]) => void }) => {
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleChange = (event: ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(event.target.files ?? []);
    event.target.value = "";
    if (files.length === 0) return;
    onFiles(files);
  };

  return (
    <>
      <input
        ref={fileInputRef}
        type="file"
        accept="image/*"
        multiple
        className="sr-only"
        aria-label="Choose image files"
        onChange={handleChange}
      />
      <Button
        type="button"
        size="icon"
        variant="ghost"
        aria-label="Attach image"
        className="mt-0.5"
        onClick={() => fileInputRef.current?.click()}
      >
        <ImagePlus className="size-4" aria-hidden />
      </Button>
    </>
  );
};
