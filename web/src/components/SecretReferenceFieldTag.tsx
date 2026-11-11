import { useWatch, type Control, type FieldValues, type Path } from "react-hook-form";

import { SecretReferenceTag } from "@/components/SecretReferenceTag";

interface SecretReferenceFieldTagProps<T extends FieldValues> {
  control: Control<T>;
  name: Path<T>;
}

// The tag for a form field's current value.
export const SecretReferenceFieldTag = <T extends FieldValues>({ control, name }: SecretReferenceFieldTagProps<T>) => {
  const value: unknown = useWatch({ control, name });
  return typeof value === "string" && <SecretReferenceTag value={value} />;
};
