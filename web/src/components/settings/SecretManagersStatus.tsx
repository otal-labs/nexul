import { secretManagerName } from "@nexul/client-core/secretReference";

import { SettingsStatus, type SettingsStatusTone } from "@/components/settings/SettingsStatus";
import { useFetchSecretManagers } from "@/hooks/ConnectorsHooks";
import type { SecretManagerState } from "@/models/Connectors";

const toneByState: Record<SecretManagerState, SettingsStatusTone> = {
  ready: "success",
  "token rejected": "destructive",
  unreachable: "warning",
};

// The server's own secrets managers, one dot and name each; a rejected token or an outage says why beside the name.
export const SecretManagersStatus = () => {
  const { data } = useFetchSecretManagers();

  if (!data?.length) return null;

  return (
    <ul aria-label="Secrets managers" className="flex flex-wrap items-center gap-x-4 gap-y-1">
      {data.map((manager) => (
        <li key={manager.manager}>
          <SettingsStatus tone={toneByState[manager.state]} detail={manager.state === "ready" ? undefined : manager.state}>
            {secretManagerName(manager.manager)}
          </SettingsStatus>
        </li>
      ))}
    </ul>
  );
};
