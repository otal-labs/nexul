import type { ComponentProps } from "react";

import type { ActionGate } from "@nexul/client-core/actionState";

import { ActionButton } from "@/components/action/ActionButton";
import { SettingsSaveBar } from "@/components/settings/SettingsSaveBar";
import { useActionState } from "@/hooks/useActionState";

type GatedSaveBarProps = ComponentProps<typeof SettingsSaveBar> & ActionGate;

// The card footer's Save for a form only some roles may change: the strip when they may, a locked Save when a role shows it.
export const GatedSaveBar = ({ permission, projectId, answer, verb, ...bar }: GatedSaveBarProps) => {
  const action = useActionState({ permission, projectId, answer, verb });
  return (
    <>
      {action.state === "live" && <SettingsSaveBar {...bar} />}
      {action.state === "locked" && (
        <ActionButton permission={permission} projectId={projectId} answer={answer} verb={verb} size="sm">
          {bar.saveLabel ?? "Save"}
        </ActionButton>
      )}
    </>
  );
};
