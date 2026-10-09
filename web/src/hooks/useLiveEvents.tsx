import { useEffect, useRef } from "react";
import { type QueryClient, useQueryClient } from "@tanstack/react-query";
import { useLocation, useNavigate } from "react-router";

import type { MyWorkspaceInfo } from "@nexul/client-core/permissions";
import { LiveEventsClient, type LiveEventsClientOptions, type ServerFrame } from "@nexul/client-core/liveSocket";

import { authFollower, getMeKey } from "@/hooks/AuthHooks";
import { botwebhookFollower } from "@/hooks/BotwebhookHooks";
import { categoryFollower } from "@/hooks/CategoryHooks";
import { chatFollower } from "@/hooks/ChatFollower";
import { computerSetupFollower } from "@/hooks/ComputerSetupHooks";
import { deployFollower } from "@/hooks/DeployHooks";
import { dnsFollower } from "@/hooks/DnsHooks";
import { docFolderFollower } from "@/hooks/DocFolderHooks";
import { docFollower } from "@/hooks/DocHooks";
import { instanceUpgradeFollower } from "@/hooks/InstanceUpgradeHooks";
import { interviewSourceFollower } from "@/hooks/InterviewSourceHooks";
import { memoryFollower } from "@/hooks/MemoryHooks";
import { noteFollower } from "@/hooks/NoteHooks";
import { notificationFollower } from "@/hooks/NotificationHooks";
import { pairingFollower } from "@/hooks/PairingHooks";
import { peopleFollower } from "@/hooks/PeopleHooks";
import { playFollower } from "@/hooks/PlayHooks";
import { projectFollower } from "@/hooks/ProjectHooks";
import { reactionFollower } from "@/hooks/ReactionHooks";
import { roleFollower } from "@/hooks/RoleHooks";
import { runnerFollower } from "@/hooks/RunnerHooks";
import { serviceFollower } from "@/hooks/ServiceHooks";
import { stackFollower } from "@/hooks/StackHooks";
import { statusFollower } from "@/hooks/StatusHooks";
import { teamFollower } from "@/hooks/TeamHooks";
import { templateFollower } from "@/hooks/TemplateHooks";
import { ticketFollower } from "@/hooks/TicketHooks";
import { ticketLinkFollower } from "@/hooks/TicketLinkHooks";
import { ticketTypeFollower } from "@/hooks/TicketTypeHooks";
import { topologyFollower } from "@/hooks/TopologyHooks";
import { trailFollower } from "@/hooks/TrailHooks";
import { notifyIfServerUpdated } from "@/hooks/VersionHooks";
import { voiceFollower } from "@/hooks/VoiceHooks";
import { getMyRoleKey, workspaceFollower } from "@/hooks/WorkspaceHooks";
import type { MeResponse } from "@/models/User";
import { followEach, type Live, type LiveFollower } from "@/lib/live";

const heldPermissions = (client: QueryClient, workspaceId: string) =>
  JSON.stringify([client.getQueryData<MyWorkspaceInfo>([getMyRoleKey, workspaceId]), client.getQueryData<MeResponse>([getMeKey])?.instance_permissions]);

// Every read is checked on the server, so once the viewer's permissions move, all open reads refetch through those checks.
const permissionFollower: LiveFollower = {
  ...followEach(
    ["workspace.member.added", "workspace.member.removed", "workspace.member.updated", "access.grant.changed"],
    ({ user_id: userID }: { user_id?: string }, { client }) =>
      !!userID && userID === client.getQueryData<MeResponse>([getMeKey])?.user.id && client.invalidateQueries(),
  ),
  // A role frame names no holder, so the viewer's own permissions in that workspace are refetched and compared.
  "role.updated": async ({ workspace_id: workspaceId }: { workspace_id: string }, { client }) => {
    const before = heldPermissions(client, workspaceId);
    await Promise.all([client.refetchQueries({ queryKey: [getMyRoleKey, workspaceId], exact: true }), client.refetchQueries({ queryKey: [getMeKey] })]);
    if (heldPermissions(client, workspaceId) === before) return;
    await client.invalidateQueries();
  },
};

// Each domain follows its own topics next to its keys (ADR 0134); the socket only routes.
const followers: LiveFollower[] = [
  authFollower,
  botwebhookFollower,
  categoryFollower,
  chatFollower,
  computerSetupFollower,
  deployFollower,
  dnsFollower,
  docFolderFollower,
  docFollower,
  instanceUpgradeFollower,
  interviewSourceFollower,
  memoryFollower,
  noteFollower,
  notificationFollower,
  pairingFollower,
  peopleFollower,
  permissionFollower,
  playFollower,
  projectFollower,
  reactionFollower,
  roleFollower,
  runnerFollower,
  serviceFollower,
  stackFollower,
  statusFollower,
  teamFollower,
  templateFollower,
  ticketFollower,
  ticketLinkFollower,
  ticketTypeFollower,
  topologyFollower,
  trailFollower,
  voiceFollower,
  workspaceFollower,
];

const routes = new Map<string, LiveFollower[string][]>();
for (const follower of followers) {
  for (const [topic, follow] of Object.entries(follower)) routes.set(topic, [...(routes.get(topic) ?? []), follow]);
}

export const followedTopics: ReadonlySet<string> = new Set(routes.keys());

const dispatch = (live: () => Live) => (frame: ServerFrame) => {
  const follows = routes.get(frame.topic);
  if (!follows) return;
  const context = live();
  for (const follow of follows) void follow(frame.payload as never, context);
};

export const useLiveEvents = (url: string | null, opts: LiveEventsClientOptions = {}) => {
  const client = useQueryClient();
  const optsRef = useRef(opts);
  const navigate = useNavigate();
  const location = useLocation();
  // The socket outlives every render, so a follower reads the router's latest state from here.
  const routerRef = useRef({ navigate, location });
  useEffect(() => {
    routerRef.current = { navigate, location };
  });

  useEffect(() => {
    if (!url) return;
    const events = new LiveEventsClient(url, {
      ...optsRef.current,
      // Frames sent while the socket was down are lost, so every open read refetches, as after an access change.
      onReconnect: () => {
        optsRef.current.onReconnect?.();
        void notifyIfServerUpdated(client);
        void client.invalidateQueries();
      },
    });
    const unsubscribe = events.subscribe(dispatch(() => ({ client, ...routerRef.current })));
    events.connect();
    return () => {
      unsubscribe();
      events.close();
    };
  }, [url, client]);
};
