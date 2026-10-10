import { beforeEach, describe, expect, it, vi } from "vitest";

import { useFetchMe } from "@/hooks/AuthHooks";
import { useFetchBotwebhooks } from "@/hooks/BotwebhookHooks";
import { useFetchCategories, useFetchProjectCategories } from "@/hooks/CategoryHooks";
import { useFetchChatUnread, useFetchConversations, useFetchMessages } from "@/hooks/ChatHooks";
import { useFetchComputerSetup } from "@/hooks/ComputerSetupHooks";
import { useFetchDeploy, useFetchDeployLog } from "@/hooks/DeployHooks";
import { useFetchDoc, useFetchDocClarification, useFetchDocs, useFetchDocsByProject } from "@/hooks/DocHooks";
import { useFetchDocFolders } from "@/hooks/DocFolderHooks";
import { useFetchProjectAccess } from "@/hooks/ProjectHooks";
import { useFetchInterviewSources } from "@/hooks/InterviewSourceHooks";
import { useFetchMemories, useFetchMemoriesByProject, useFetchMemory, useFetchMemoryVersions } from "@/hooks/MemoryHooks";
import { useFetchInbox, useFetchUnreadByWorkspace, useFetchUnreadCount } from "@/hooks/NotificationHooks";
import { useFetchProjectPeople, useFetchWorkspacePeople } from "@/hooks/PeopleHooks";
import { useFetchApplicablePlays, useFetchWorkspacePlays } from "@/hooks/PlayHooks";
import { useFetchWorkspaceRoles } from "@/hooks/RoleHooks";
import { useRunnerQueue, useRunners } from "@/hooks/RunnerHooks";
import { useFetchStackDeploys } from "@/hooks/StackHooks";
import { useFetchProjectStatuses } from "@/hooks/StatusHooks";
import { useFetchTeam } from "@/hooks/TeamHooks";
import { useFetchTicket, useFetchTicketsByProject } from "@/hooks/TicketHooks";
import { useFetchBlockers, useFetchTicketLinkSet } from "@/hooks/TicketLinkHooks";
import { useFetchActiveTrails } from "@/hooks/TrailHooks";
import { useFetchMyRole, useFetchWorkspaces } from "@/hooks/WorkspaceHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { mountLive } from "@/test/liveHarness";

vi.mock("@/api/client", () => ({ api: { get: vi.fn() }, errorMessage: vi.fn() }));

// What the server answers for the views below; any list not named here is empty.
const server: Record<string, unknown> = {
  "/api/auth/me": { user: { id: "u-me" }, instance_permissions: [] },
  "/api/chat/conversations": [{ id: "c-1", workspace_id: "ws-1", kind: "channel", name: "eng", created_by: "u-me", created_at: "", updated_at: "" }],
  "/api/categories": [{ id: "cat-1", project_id: "p-1", name: "Payments", position: 0, color: "", created_at: "", updated_at: "" }],
  "/api/chat/unread": {},
  "/api/deploys/d-1": { id: "d-1", stack_id: "s-1", status: "healthy" },
  "/api/stacks/s-1/deploys": [{ id: "d-1", stack_id: "s-1", status: "healthy" }],
  "/api/runners": [{ id: "r-1", name: "box", connected: true, last_seen: "", running_job: { id: "d-1", kind: "build" }, version: "" }],
  "/api/docs/d-1": { id: "d-1", project_id: "p-1", folder_id: "f-1", title: "Spec", version: 1 },
  "/api/docs": [{ id: "d-1", project_id: "p-1", folder_id: "f-1", title: "Spec", version: 1 }],
  "/api/docs/d-1/clarification": { rounds: [] },
  "/api/memories/m-1": { id: "m-1", project_id: "p-1", title: "Deploys" },
  "/api/memories": [{ id: "m-1", project_id: "p-1", title: "Deploys" }],
  "/api/statuses": [{ id: "st-1", name: "Doing", position: 0, kind: "progress", icon: "", created_at: "", updated_at: "" }],
  "/api/tickets/t-1/ticket-links": {
    found_in: null,
    origin_unknown: false,
    bugs_found: [],
    blocked_by: [{ id: "t-2", project_id: "p-1", prefix: "ACME", number: 2, title: "Login", status: "st-1", done: false }],
    blocks: [],
    blocked: true,
  },
  "/api/tickets": [{ id: "t-1", project_id: "p-1", doc_id: "", category_id: "cat-1", status: "st-1", position: 0, title: "Login", updated_at: "" }],
  "/api/tickets/t-1": { id: "t-1", project_id: "p-1", doc_id: "", category_id: "cat-1", status: "st-1", position: 0, title: "Login", updated_at: "" },
  "/api/projects/p-1/access": { access: [] },
  "/api/projects/p-2/access": { access: [] },
  "/api/projects/p-1/people": { people: [] },
  "/api/projects/p-2/people": { people: [] },
  "/api/tickets/blockers": {},
  "/api/workspaces/ws-1/people": { people: [{ user_id: "u-2", login: "lena", display_name: "Lena", avatar_url: "" }] },
  "/api/workspaces/ws-1/me": { role_name: "Editor", permissions: [] },
  "/api/team": { people: [], roles: [], workspaces: [] },
};

const respond = (url: string) => server[url] ?? [];

// Each view stands for a page someone has open while the frame arrives.
const view = (useHooks: () => void) => {
  const View = () => {
    useHooks();
    return null;
  };
  return <View />;
};

describe("requests a live frame sends", () => {
  beforeEach(() => {
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  });

  it("refetches only the unread counts when a message lands in an open conversation", async () => {
    const { requestsAfter } = await mountLive(
      view(() => {
        useFetchConversations("ws-1");
        useFetchChatUnread("ws-1");
        useFetchMessages("c-1");
      }),
      respond,
    );
    const message = { id: "m-9", conversation_id: "c-1", author_id: "u-2", author_kind: "user", body: "hi", mentions: null, created_at: "", updated_at: "" };
    expect(await requestsAfter("chat.message.created", { message, workspace_id: "ws-1" })).toEqual(["/api/chat/unread?workspace_id=ws-1"]);
  });

  it("refetches only the message's workspace's list and unread counts for a conversation no list holds yet", async () => {
    const { requestsAfter } = await mountLive(
      view(() => {
        useFetchConversations("ws-1");
        useFetchChatUnread("ws-1");
        useFetchConversations("ws-2");
        useFetchChatUnread("ws-2");
      }),
      respond,
    );
    const message = { id: "m-9", conversation_id: "dm-new", author_id: "u-2", author_kind: "user", body: "hi", mentions: null, created_at: "", updated_at: "" };
    expect((await requestsAfter("chat.message.created", { message, workspace_id: "ws-2" })).sort()).toEqual([
      "/api/chat/conversations?workspace_id=ws-2",
      "/api/chat/unread?workspace_id=ws-2",
    ]);
  });

  it("refetches the unread counts of every workspace whose list holds a DM, and no list", async () => {
    const dm = { id: "dm-1", workspace_id: "ws-1", kind: "dm", name: "", participant_ids: ["u-me", "u-2"], created_by: "u-me", created_at: "", updated_at: "" };
    const { requestsAfter } = await mountLive(
      view(() => {
        useFetchConversations("ws-1");
        useFetchChatUnread("ws-1");
        useFetchConversations("ws-2");
        useFetchChatUnread("ws-2");
      }),
      (url) => (url === "/api/chat/conversations" ? [dm] : respond(url)),
    );
    const message = { id: "m-9", conversation_id: "dm-1", author_id: "u-2", author_kind: "user", body: "hi", mentions: null, created_at: "", updated_at: "" };
    expect((await requestsAfter("chat.message.created", { message, workspace_id: "ws-1" })).sort()).toEqual([
      "/api/chat/unread?workspace_id=ws-1",
      "/api/chat/unread?workspace_id=ws-2",
    ]);
  });

  it("refetches only the unread counts of a deleted message's workspace", async () => {
    const { requestsAfter } = await mountLive(
      view(() => {
        useFetchChatUnread("ws-1");
        useFetchChatUnread("ws-2");
      }),
      respond,
    );
    const deleted = { conversation_id: "c-1", message_id: "m-1", deleted_at: "", workspace_id: "ws-1" };
    expect(await requestsAfter("chat.message.deleted", deleted)).toEqual(["/api/chat/unread?workspace_id=ws-1"]);
  });

  it("drops a deleted conversation from the list without refetching it", async () => {
    const { requestsAfter } = await mountLive(
      view(() => {
        useFetchConversations("ws-1");
        useFetchChatUnread("ws-1");
      }),
      respond,
    );
    const deleted = { conversation_id: "c-1", workspace_id: "ws-1", kind: "channel", name: "eng" };
    expect(await requestsAfter("chat.conversation.deleted", deleted)).toEqual(["/api/chat/unread?workspace_id=ws-1"]);
  });

  it("refetches only the history of the stack a new deploy names, and leaves an open deploy and its log alone", async () => {
    const { requestsAfter } = await mountLive(
      view(() => {
        useFetchDeploy("d-1");
        useFetchDeployLog("d-1");
        useFetchStackDeploys("s-1");
        useFetchStackDeploys("s-2");
      }),
      respond,
    );
    expect(await requestsAfter("deploy.updated", { id: "d-9", status: "pending", stack_id: "s-2" })).toEqual(["/api/stacks/s-2/deploys"]);
  });

  it("refetches an open deploy and its log, not its stack's history, for a log batch", async () => {
    const { requestsAfter } = await mountLive(
      view(() => {
        useFetchDeploy("d-1");
        useFetchDeployLog("d-1");
        useFetchStackDeploys("s-1");
      }),
      respond,
    );
    expect((await requestsAfter("deploy.updated", { id: "d-1", status: "healthy", stack_id: "s-1" })).sort()).toEqual(["/api/deploys/d-1", "/api/deploys/d-1/log"]);
  });

  it.each(["deploy.build_progress", "deploy.deploy_progress"])("sends nothing on %s for a job the runners list already shows", async (topic) => {
    const { requestsAfter } = await mountLive(
      view(() => {
        useRunners();
        useRunnerQueue();
      }),
      respond,
    );
    expect(await requestsAfter(topic, { id: "d-1", step: 2, total: 5, log: "step 2" })).toEqual([]);
  });

  it("moves a doc to another folder without a request when no inbox row names it", async () => {
    const { requestsAfter } = await mountLive(
      view(() => {
        useFetchInbox();
        useFetchDocs();
        useFetchDocsByProject("p-1");
        useFetchDoc("d-1");
        useFetchDocFolders("p-1");
      }),
      respond,
    );
    const moved = { id: "d-1", project_id: "p-1", folder_id: "f-2", title: "Spec", body: "{}", version: 1, archived: false, locked: false, created_by: "", created_at: "", updated_at: "" };
    expect(await requestsAfter("doc.moved", { doc: moved, from_folder_id: "f-1" })).toEqual([]);
  });

  it("sends nothing for a clarification round on a doc nobody has open", async () => {
    const { requestsAfter } = await mountLive(view(() => useFetchDocClarification("d-1")), respond);
    expect(await requestsAfter("doc.clarification.round_started", { doc: { id: "d-2", project_id: "p-1", title: "Other" }, round: 1 })).toEqual([]);
  });

  it("sends nothing for a play changed in another workspace", async () => {
    const { requestsAfter } = await mountLive(
      view(() => {
        useFetchWorkspacePlays("ws-1");
        useFetchApplicablePlays("ws-1", "p-1", "ticket", undefined);
      }),
      respond,
    );
    const play = { id: "pl-9", workspace_id: "ws-2", label: "Review" };
    expect(await requestsAfter("play.updated", { play })).toEqual([]);
  });

  it("refetches only the workspace's memory list for a memory in another project", async () => {
    const { requestsAfter } = await mountLive(
      view(() => {
        useFetchMemories("ws-1");
        useFetchMemoriesByProject("p-1");
        useFetchMemory("m-1");
        useFetchMemoryVersions("m-1");
        useFetchInterviewSources("p-1");
      }),
      respond,
    );
    const memory = { id: "m-9", workspace_id: "ws-1", project_id: "p-2", title: "Other", when_to_use: "", always_included: false, footer: false, version: 2, updated_at: "" };
    expect(await requestsAfter("memory.updated", { memory, author_id: "u-2" })).toEqual(["/api/memories?workspace_id=ws-1"]);
  });

  it("renames a board column without a request", async () => {
    const { requestsAfter } = await mountLive(
      view(() => {
        useFetchTicketsByProject("p-1");
        useFetchProjectStatuses("p-1");
        useFetchTicketLinkSet("t-1");
        useFetchBlockers();
      }),
      respond,
    );
    const status = { id: "st-1", project_id: "p-1", name: "Building", position: 0, kind: "progress", icon: "", created_at: "", updated_at: "" };
    expect(await requestsAfter("status.updated", { status, previous_kind: "progress" })).toEqual([]);
  });

  it("leaves the link views alone when a column whose board is not open is renamed", async () => {
    const { requestsAfter } = await mountLive(
      view(() => {
        useFetchTicketLinkSet("t-1");
        useFetchBlockers();
      }),
      respond,
    );
    const status = { id: "st-1", project_id: "p-1", name: "Building", position: 0, kind: "progress", icon: "", created_at: "", updated_at: "" };
    expect(await requestsAfter("status.updated", { status, previous_kind: "progress" })).toEqual([]);
    expect((await requestsAfter("status.updated", { status: { ...status, kind: "done" }, previous_kind: "progress" })).sort()).toEqual([
      "/api/tickets/blockers",
      "/api/tickets/t-1/ticket-links",
    ]);
  });

  it("moves a ticket to another category without a request", async () => {
    const { requestsAfter } = await mountLive(
      view(() => {
        useFetchTicketsByProject("p-1");
        useFetchTicket("t-1");
      }),
      respond,
    );
    expect(await requestsAfter("ticket.category_changed", { ticket_id: "t-1", category_id: "cat-2", project_id: "p-1" })).toEqual([]);
  });

  it("refetches the inbox and badge of a new notice's workspace and the count across workspaces, nothing for another workspace's", async () => {
    const { requestsAfter } = await mountLive(
      view(() => {
        useFetchInbox();
        useFetchUnreadCount();
        useFetchUnreadByWorkspace();
      }),
      respond,
    );
    expect((await requestsAfter("notification.created", { user_ids: ["u-me"], workspace_id: "ws-1", project_id: "p-1" })).sort()).toEqual([
      "/api/notifications/unread-count",
      "/api/notifications/unread-count?workspace_id=ws-1",
      "/api/notifications?workspace_id=ws-1",
    ]);
    expect(await requestsAfter("notification.created", { user_ids: ["u-me"], workspace_id: "ws-2" })).toEqual(["/api/notifications/unread-count"]);
  });

  it("refetches only the active runs of the project a run's state moved in", async () => {
    const { requestsAfter } = await mountLive(
      view(() => {
        useFetchActiveTrails("p-1");
        useFetchActiveTrails("p-2");
      }),
      respond,
    );
    const run = { trail_id: "tr-1", play_id: "pl-1", target_type: "ticket", target_id: "t-9", state: "running", activity: null, ended_at: null, last_error: "", project_id: "p-2", workspace_id: "ws-1" };
    expect(await requestsAfter("play.run", run)).toEqual(["/api/plays/runs/active?target_type=ticket&project_id=p-2"]);
  });

  it("refetches only its workspace's memory list once an interview run finishes", async () => {
    const { requestsAfter } = await mountLive(
      view(() => {
        useFetchMemories("ws-1");
        useFetchMemories("ws-2");
      }),
      respond,
    );
    const run = { trail_id: "tr-1", play_id: "pl-1", target_type: "interview", target_id: "p-1", state: "done", activity: null, ended_at: null, last_error: "", project_id: "p-1", workspace_id: "ws-1" };
    expect(await requestsAfter("play.run", run)).toEqual(["/api/memories?workspace_id=ws-1"]);
  });

  it("refetches who may open only the projects a membership change names", async () => {
    const { requestsAfter } = await mountLive(
      view(() => {
        useFetchProjectAccess("p-1");
        useFetchProjectAccess("p-2");
        useFetchProjectPeople("p-1");
        useFetchProjectPeople("p-2");
      }),
      respond,
    );
    expect((await requestsAfter("workspace.member.updated", { user_id: "u-2", workspace_id: "ws-1", project_ids: ["p-1"] })).sort()).toEqual([
      "/api/projects/p-1/access",
      "/api/projects/p-1/people",
    ]);
  });

  it("renames a category without a request", async () => {
    const { requestsAfter } = await mountLive(
      view(() => {
        useFetchCategories();
        useFetchProjectCategories("p-1");
      }),
      respond,
    );
    const category = { id: "cat-1", project_id: "p-1", name: "Billing", position: 0, color: "", created_at: "", updated_at: "" };
    expect(await requestsAfter("category.updated", { category })).toEqual([]);
  });

  it("refetches the people and team, not the viewer's own account, when someone else renames themselves", async () => {
    const { requestsAfter } = await mountLive(
      view(() => {
        useFetchMe();
        useFetchWorkspacePeople("ws-1");
        useFetchTeam();
      }),
      respond,
    );
    expect((await requestsAfter("account.profile_updated", { account_id: "u-2" })).sort()).toEqual(["/api/team", "/api/workspaces/ws-1/people"]);
  });

  it("refetches only the team when someone joins another workspace", async () => {
    const { requestsAfter } = await mountLive(
      view(() => {
        useFetchMe();
        useFetchWorkspaces();
        useFetchWorkspacePeople("ws-1");
        useFetchTeam();
      }),
      respond,
    );
    expect(await requestsAfter("workspace.member.added", { user_id: "u-3", workspace_id: "ws-2" })).toEqual(["/api/team"]);
  });

  it("refetches only the viewer's instance permissions when a role changes in another workspace", async () => {
    const { requestsAfter } = await mountLive(
      view(() => {
        useFetchMe();
        useFetchMyRole("ws-1");
        useFetchWorkspaceRoles("ws-1");
      }),
      respond,
    );
    expect(await requestsAfter("role.updated", { role_id: "r-9", workspace_id: "ws-2" })).toEqual(["/api/auth/me"]);
  });

  it("sends nothing when another computer's setup turn moves", async () => {
    const { requestsAfter } = await mountLive(view(() => useFetchComputerSetup("c-1")), respond);
    expect(await requestsAfter("computer.setup_turn_changed", { computer_id: "c-2", user_id: "u-me", state: "running" })).toEqual([]);
  });

  it("sends nothing when a bot changes in another conversation", async () => {
    const { requestsAfter } = await mountLive(view(() => useFetchBotwebhooks("c-1")), respond);
    expect(await requestsAfter("botwebhook.updated", { botwebhook_id: "b-9", conversation_id: "c-2", workspace_id: "ws-1", name: "ci" })).toEqual([]);
  });

  it("sends nothing when another project's interview sources change", async () => {
    const { requestsAfter } = await mountLive(view(() => useFetchInterviewSources("p-1")), respond);
    expect(await requestsAfter("interview_source.changed", { workspace_id: "ws-1", project_id: "p-2", source_id: "src-9" })).toEqual([]);
  });
});
