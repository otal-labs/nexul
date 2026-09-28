import * as Network from "expo-network";

type SetOnline = (online: boolean) => void;

// Queries pause while offline and resume on reconnect, per the TanStack React Native setup.
export const networkOnlineListener = (setOnline: SetOnline) => {
  let initialised = false;
  const subscription = Network.addNetworkStateListener((state) => {
    initialised = true;
    setOnline(state.isConnected === true);
  });
  Network.getNetworkStateAsync()
    .then((state) => {
      if (initialised) return;
      setOnline(state.isConnected === true);
    })
    .catch(() => undefined);
  return () => subscription.remove();
};
