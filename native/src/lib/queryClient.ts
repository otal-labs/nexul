import { QueryClient, onlineManager } from "@tanstack/react-query";

import { networkOnlineListener } from "@/lib/onlineStatus";

onlineManager.setEventListener(networkOnlineListener);

export const queryClient = new QueryClient();
