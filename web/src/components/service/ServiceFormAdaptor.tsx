import { useFormContext, useWatch } from "react-hook-form";

import { FormInput } from "@/components/FormInput";
import { DeployStrategy, type ServiceFormData } from "@/models/Service";

export const ServiceFormAdaptor = () => {
  const { control } = useFormContext<ServiceFormData>();
  const strategy = useWatch({ control, name: "strategy" });

  if (strategy === DeployStrategy.Compose) {
    return (
      <FormInput control={control} name="compose_dir" label="Compose directory" placeholder="/srv/api" />
    );
  }

  return <FormInput control={control} name="docker_network" label="Docker network" placeholder="app-net" />;
};
