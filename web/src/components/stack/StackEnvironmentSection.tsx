import { useState } from "react";

import { StackEnvironmentForm } from "@/components/stack/StackEnvironmentForm";
import { StackEnvironmentKeys } from "@/components/stack/StackEnvironmentKeys";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useFlash } from "@/hooks/useFlash";
import type { Stack } from "@/models/Stack";

interface StackEnvironmentSectionProps {
  stack: Stack;
  // The image this stack currently runs, for Deploy now.
  image: string | undefined;
}

// Someone who can change the stack edits its variables, values hidden until revealed; everyone else is sent the keys only.
export const StackEnvironmentSection = ({ stack, image }: StackEnvironmentSectionProps) => {
  const can = useAreaAccess(stack.project_id || undefined);
  const [revealed, setRevealed] = useState(false);
  const [saved, flash] = useFlash();
  const keys = Object.keys(stack.env ?? {}).sort();
  const derived = !!stack.derived_from;
  const editable = !derived && !stack.env_masked && (can?.("editStacks") ?? false);

  return (
    <>
      {derived && (
        <StackEnvironmentKeys keys={keys} description="A branch deployment takes its variables from the base stack and the branch rule's overrides." />
      )}
      {!derived && !editable && (
        <StackEnvironmentKeys keys={keys} description="You can see which variables are set. Their values are hidden from you." />
      )}
      {editable && (
        <StackEnvironmentForm
          key={stack.updated_at}
          stack={stack}
          image={image}
          revealed={revealed}
          onReveal={setRevealed}
          saved={saved}
          onSaved={flash}
        />
      )}
    </>
  );
};
