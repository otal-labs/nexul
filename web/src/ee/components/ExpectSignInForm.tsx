import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { FormInput } from "@/components/FormInput";
import { SettingsHint } from "@/components/settings/SettingsHint";
import { useExpectSignIn } from "@/ee/hooks/SignInMoveHooks";
import type { ExpectSignInFormData } from "@/ee/models/SignInMove";

interface ExpectSignInFormProps {
  accountId: string;
  providerName: string;
}

// The email the person's first sign-in through the provider will bring; a refused one (another account expects it) shows on the field.
export const ExpectSignInForm = ({ accountId, providerName }: ExpectSignInFormProps) => {
  const { control, onSubmit } = useFormDialogContext<ExpectSignInFormData>();
  const expect = useExpectSignIn();

  onSubmit(async (input) => {
    await expect.mutateAsync({ accountId, email: input.email });
    return input;
  });

  return (
    <div className="space-y-2">
      <FormInput control={control} name="email" label="Email" type="email" autoComplete="off" />
      <SettingsHint>
        Their first sign-in through {providerName} with this email opens this account instead of making a new one, whatever other account has the
        same email. It is used up then.
      </SettingsHint>
    </div>
  );
};
