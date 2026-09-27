import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { FormInput } from "@/components/FormInput";
import { Button } from "@/components/ui/button";
import { OwnHttpsFormSchema, type OwnHttpsFormData } from "@/models/Setup";

interface SetupOwnHttpsFormProps {
  onFinish: (url: string) => void;
}

export const SetupOwnHttpsForm = ({ onFinish }: SetupOwnHttpsFormProps) => {
  const form = useForm<OwnHttpsFormData>({
    defaultValues: { url: "" },
    resolver: zodResolver(OwnHttpsFormSchema),
  });

  return (
    <form onSubmit={form.handleSubmit(({ url }) => onFinish(url.replace(/\/+$/, "")))} className="space-y-4">
      <FormInput
        control={form.control}
        name="url"
        label="HTTPS address"
        placeholder="https://deploy.example.com"
        inputMode="url"
        autoComplete="off"
      />
      <p className="text-sm text-muted-foreground">
        Your proxy must forward to{" "}
        <code className="break-all rounded bg-muted px-1.5 py-0.5 font-mono text-xs text-foreground">
          http://{window.location.host}
        </code>
      </p>
      <Button type="submit" className="w-full sm:w-auto">
        Check address
      </Button>
    </form>
  );
};
