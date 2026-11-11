import { useRef, type ChangeEvent } from "react";

import { ActionButton } from "@/components/action/ActionButton";

interface BrandingFileButtonProps {
  label: string;
  accept: string;
  verb: string;
  disabled?: boolean;
  variant?: "default" | "outline";
  busy?: boolean;
  onFile: (file: File) => void;
}

// A button that opens the file picker; the file is handed over once chosen, so choosing is saving.
export const BrandingFileButton = ({ label, accept, verb, disabled = false, variant = "outline", busy = false, onFile }: BrandingFileButtonProps) => {
  const input = useRef<HTMLInputElement>(null);
  const onChange = (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.currentTarget.files?.[0];
    e.currentTarget.value = "";
    if (file) onFile(file);
  };
  return (
    <>
      <ActionButton type="button" permission="instance:write" verb={verb} size="sm" variant={variant} className="disabled:opacity-50" disabled={disabled} loading={busy} onClick={() => input.current?.click()}>
        {label}
      </ActionButton>
      <input ref={input} type="file" accept={accept} aria-label={label} hidden onChange={onChange} />
    </>
  );
};
