import { act, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { toast } from "sonner";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useWorkspaceStore } from "@/stores/workspaceStore";
import { computer, pairedBefore, renderComputers, unconfirmed } from "@/test/agentComputers";
import { asOwner } from "@/test/viewer";

const mocks = vi.hoisted(() => ({ get: vi.fn(), put: vi.fn(), post: vi.fn(), patch: vi.fn(), del: vi.fn(), errorMessage: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, put: mocks.put, post: mocks.post, patch: mocks.patch, delete: mocks.del },
  errorMessage: mocks.errorMessage,
}));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));
vi.mock("@/components/settings/ComputerMCPToken", () => ({ ComputerMCPToken: () => null }));

let setup: Record<string, unknown> = unconfirmed;
let presence: Record<string, string> = {};
let computers: unknown[] = [];
let extra: (url: string) => unknown = () => undefined;

// Every read the page makes, with the computers as the list and the open pane both see them.
const rest = (url: string) => {
  const own = extra(url);
  if (own !== undefined) return own;
  if (url === "/api/pairing/computers") return { computers };
  if (url === "/api/pairing/presence") return { computers: presence };
  if (url === "/api/runners/enrollments") return [];
  if (url === "/api/pairing/defaults") return {};
  if (url === "/api/pairing/projects") return { links: [] };
  if (url === "/api/projects") return [];
  if (url === "/api/workspaces") return [{ id: "ws-1", name: "Acme", slug: "acme" }];
  if (url.endsWith("/setup")) return setup;
  if (url.endsWith("/providers")) return { providers: [] };
  throw new Error(`unexpected GET ${url}`);
};

const serveComputers = (list: unknown[]) => {
  computers = list;
};

const openComputer = (id = "c1") => `/acme/agent?computers=${id}`;
const pane = (name: string) => screen.findByRole("heading", { name, level: 3 });
// The open computer's pane, apart from the list beside it, which repeats each computer's state.
const details = () => within(screen.getByRole("region", { name: "Details" }));
const yourComputers = () => screen.getByRole("region", { name: "Your computers" });
const goTo = async (user: ReturnType<typeof userEvent.setup>, name: string) => {
  await user.click(within(yourComputers()).getByRole("link", { name: new RegExp(`^${name}`) }));
  await pane(name);
};

describe("the computers on the Agent page", () => {
  beforeEach(() => {
    for (const fn of Object.values(mocks)) fn.mockReset();
    mocks.get.mockImplementation(asOwner(rest));
    vi.mocked(toast.success).mockClear();
    vi.mocked(toast.error).mockClear();
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedWorkspaceSlug: "acme" });
    useWorkspaceStore.persist.clearStorage();
    setup = unconfirmed;
    presence = {};
    computers = [];
    extra = () => undefined;
  });

  it("lists a computer someone shares with you under yours, by its owner's name and online state only", async () => {
    serveComputers([
      pairedBefore(),
      { id: "c-bob", name: "Laptop", runner: { connected: true }, shared: { owner_id: "u-bob", owner_name: "Bob", permissions: ["computers:run_commands", "computers:see"] } },
    ]);
    renderComputers(openComputer("c-bob"));

    const group = await screen.findByRole("region", { name: "Shared with you" });
    expect(within(group).getByRole("link", { name: /Bob's Laptop/ })).toHaveTextContent("Online");
    expect(await pane("Bob's Laptop")).toBeInTheDocument();
    expect(details().getByText("Online")).toBeInTheDocument();
    expect(details().getByText("Run commands")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Actions for Laptop" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Actions for Bob's Laptop" })).not.toBeInTheDocument();
  });

  it("offers adding a shared computer to your order only where its owner allows Run agents", async () => {
    const user = userEvent.setup();
    const agents = { owner_id: "u-bob", owner_name: "Bob", permissions: ["computers:see", "computers:run_agents"] };
    serveComputers([pairedBefore(), { id: "c-bob", name: "Laptop", runner: { connected: true }, position: 0, shared: agents }]);
    mocks.post.mockResolvedValue({ data: {} });
    renderComputers(openComputer("c-bob"));

    await user.click(await screen.findByRole("button", { name: "Add to your order" }));
    expect(mocks.post).toHaveBeenCalledWith("/api/pairing/computers/c-bob/order");
  });

  it("lists a shared computer you added among yours, in its place, marked shared, and takes it out again", async () => {
    const user = userEvent.setup();
    const added = { owner_id: "u-bob", owner_name: "Bob", permissions: ["computers:see", "computers:run_agents"], in_order: true };
    serveComputers([pairedBefore({ position: 1 }), { id: "c-bob", name: "Laptop", runner: { connected: true }, position: 0, shared: added }]);
    mocks.del.mockResolvedValue({ data: {} });
    renderComputers(openComputer("c-bob"));

    const yours = await screen.findByRole("region", { name: "Your computers" });
    const rows = within(yours).getAllByRole("link");
    expect(rows.map((row) => row.querySelector("[title]")?.getAttribute("title"))).toEqual(["Bob's Laptop", "Home"]);
    expect(within(rows[0]!).getByRole("img", { name: "Shared with you" })).toBeInTheDocument();
    expect(screen.queryByRole("region", { name: "Shared with you" }), "it leaves the shared group while it is in your order").not.toBeInTheDocument();

    await user.click(await screen.findByRole("button", { name: "Remove from your order" }));
    expect(mocks.del).toHaveBeenCalledWith("/api/pairing/computers/c-bob/order");
  });

  it("offers no place in your order for a share without Run agents", async () => {
    serveComputers([{ id: "c-bob", name: "Laptop", runner: { connected: true }, shared: { owner_id: "u-bob", owner_name: "Bob", permissions: ["computers:see", "computers:run_commands"] } }]);
    renderComputers(openComputer("c-bob"));

    expect(await pane("Bob's Laptop")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Add to your order" })).not.toBeInTheDocument();
  });

  it("shows a shared computer that just dropped as reconnecting, not offline", async () => {
    serveComputers([{ id: "c-bob", name: "Laptop", runner: { connected: false, reconnecting: true }, shared: { owner_id: "u-bob", owner_name: "Bob", permissions: ["computers:see"] } }]);
    renderComputers(openComputer("c-bob"));

    expect(await pane("Bob's Laptop")).toBeInTheDocument();
    expect(details().getByText("Reconnecting")).toBeInTheDocument();
    expect(screen.queryByText("Offline")).not.toBeInTheDocument();
  });

  it("shows an empty state when there are no computers", async () => {
    renderComputers("/acme/agent?computers=");
    expect(await screen.findByText("No computers yet")).toBeInTheDocument();
  });

  it("opens a computer paired before the app with its address and version in view", async () => {
    serveComputers([pairedBefore()]);
    const user = userEvent.setup();
    renderComputers("/acme/agent?computers=");

    await user.click(await within(await screen.findByRole("region", { name: "Your computers" })).findByRole("link", { name: /Home/ }));
    await pane("Home");
    expect(screen.getAllByRole("term").map((t) => t.textContent)).toContain("Address");
    expect(screen.getByText("https://home.example.com")).toBeInTheDocument();
    expect(screen.getByText("0.0.34")).toBeInTheDocument();
  });

  it("shows a computer reached through its runner by its T3 Code and facts, never the relay address", async () => {
    serveComputers([
      computer({
        server_url: "http://c1.nexul-computer.invalid",
        facts_at: "2026-10-10T12:00:00Z",
        facts: {
          hostname: "alice-laptop",
          os: "linux",
          arch: "amd64",
          git_name: "Alice Example",
          git_email: "alice@example.com",
          free_disk_bytes: 50 * 1024 ** 3,
          t3: { state: "answering", install: "service", port: 47200, version: "0.0.46" },
          providers: [{ id: "codex", driver: "codex", name: "Codex", version: "0.50.0", sign_in: "signed_out", models: [{ slug: "gpt-5", name: "GPT-5" }] }],
          projects: [{ id: "p1", title: "app", path: "/home/alice/code/app" }],
        },
      }),
    ]);
    renderComputers(openComputer());

    await pane("Home");
    expect(screen.getAllByRole("term").map((t) => t.textContent)).not.toContain("Address");
    expect(screen.queryByText(/nexul-computer\.invalid/)).not.toBeInTheDocument();
    expect(screen.getByText("Running")).toBeInTheDocument();
    expect(screen.getByText(/port 47200 · 0\.0\.46 · background service/)).toBeInTheDocument();
    expect(screen.getByText("Signed out")).toBeInTheDocument();
    expect(screen.getByText("/home/alice/code/app")).toBeInTheDocument();
    expect(screen.getByText("Alice Example <alice@example.com>")).toBeInTheDocument();
    expect(screen.getByText("50.0 GB")).toBeInTheDocument();
  });

  it("warns when a computer has no GitHub sign-in, says so quietly when it has one, and shows nothing for an older runner", async () => {
    const t3 = { state: "answering" };
    serveComputers([
      computer({ facts: { t3, github_sign_in: "none", providers: [], projects: [] } }),
      computer({ id: "c2", name: "Desk", facts: { t3, github_sign_in: "gh", providers: [], projects: [] } }),
      computer({ id: "c3", name: "Old", facts: { t3, providers: [], projects: [] } }),
    ]);
    const user = userEvent.setup();
    renderComputers(openComputer());

    await pane("Home");
    expect(screen.getByText("Not signed in")).toBeInTheDocument();
    expect(screen.getByText(/Plays can't push\. Run gh auth login\./)).toBeInTheDocument();
    await goTo(user, "Desk");
    expect(screen.getByText("Signed in")).toBeInTheDocument();
    await goTo(user, "Old");
    expect(screen.getAllByRole("term").filter((t) => t.textContent === "GitHub")).toHaveLength(0);
  });

  it("lists each tool the computer has, with its version and where it is, and the ones it lacks", async () => {
    const tools = [{ name: "git", path: "C:\\Program Files\\Git\\cmd\\git.exe", version: "2.47.1.windows.1" }, { name: "gh" }];
    serveComputers([computer({ facts: { t3: { state: "answering" }, tools, providers: [], projects: [] } })]);
    renderComputers(openComputer());

    await pane("Home");
    expect(screen.getAllByRole("term").filter((t) => t.textContent === "Tools")).toHaveLength(1);
    expect(screen.getByText("2.47.1.windows.1")).toBeInTheDocument();
    expect(screen.getByText("C:\\Program Files\\Git\\cmd\\git.exe")).toBeInTheDocument();
    expect(screen.getByText("Installed")).toBeInTheDocument();
    expect(screen.getByText("Not installed")).toBeInTheDocument();
  });

  it("lists a provider switched on whose CLI is missing as not installed, beside the installed ones", async () => {
    const providers = [{ id: "claudeAgent", driver: "claudeAgent", name: "Claude", sign_in: "signed_in" as const, models: [] }];
    const missing_providers = [{ id: "codex", driver: "codex", name: "Codex" }];
    serveComputers([computer({ facts: { t3: { state: "answering" }, providers, missing_providers, projects: [] } })]);
    renderComputers(openComputer());

    await pane("Home");
    expect(screen.getByText("Codex").closest("li")).toHaveTextContent("Not installed");
    expect(screen.getByText("Claude").closest("li")).toHaveTextContent("Signed in");
  });

  it("lists the provider CLIs row when every provider switched on is missing", async () => {
    serveComputers([computer({ facts: { t3: { state: "answering" }, providers: [], missing_providers: [{ id: "codex", driver: "codex", name: "Codex" }], projects: [] } })]);
    renderComputers(openComputer());

    await pane("Home");
    expect(screen.getAllByRole("term").filter((t) => t.textContent === "Provider CLIs")).toHaveLength(1);
    expect(screen.getByText("Codex").closest("li")).toHaveTextContent("Not installed");
  });

  it("installs a missing tool from its row once the person has seen the command it runs", async () => {
    const install = ["brew", "install", "gh"];
    serveComputers([computer({ facts: { t3: { state: "answering" }, tools: [{ name: "git", path: "/usr/bin/git", version: "2.47.1" }, { name: "gh", install }], providers: [], projects: [] } })]);
    mocks.post.mockResolvedValue({ data: undefined });
    const user = userEvent.setup();
    renderComputers(openComputer());

    await pane("Home");
    await user.click(screen.getByRole("button", { name: "Install gh" }));
    expect(await screen.findByText("Install gh on Home?")).toBeInTheDocument();
    expect(screen.getByText("brew install gh")).toBeInTheDocument();
    expect(mocks.post).not.toHaveBeenCalled();

    await user.click(screen.getAllByRole("button", { name: "Install gh" }).at(-1)!);
    await waitFor(() => expect(mocks.post).toHaveBeenCalledWith("/api/pairing/computers/c1/tools", { tool: "gh" }));
    await waitFor(() => expect(toast.success).toHaveBeenCalledWith("gh installed on Home"));
  });

  it("asks for the sudo password when sudo wants one, keeps asking while it is wrong, and sends it with that install", async () => {
    const install = ["sudo", "-n", "apt-get", "install", "-y", "git"];
    serveComputers([computer({ facts: { t3: { state: "answering" }, tools: [{ name: "git", install }], providers: [], projects: [] } })]);
    const sudoRefused = (message: string) => ({ message, response: { data: { error: message, details: { sudo_password_needed: true, computer_id: "c1" } } } });
    mocks.errorMessage.mockImplementation((error: { message: string }) => error.message);
    mocks.post
      .mockRejectedValueOnce(sudoRefused("installing git needs your sudo password"))
      .mockRejectedValueOnce(sudoRefused("that sudo password was wrong"))
      .mockResolvedValueOnce({ data: undefined });
    const user = userEvent.setup();
    renderComputers(openComputer());

    await pane("Home");
    await user.click(screen.getByRole("button", { name: "Install git" }));
    await user.click(screen.getAllByRole("button", { name: "Install git" }).at(-1)!);
    expect(await screen.findByText("Sudo password for Home")).toBeInTheDocument();
    expect(toast.error).not.toHaveBeenCalled();

    await user.type(screen.getByLabelText("Sudo password"), "hunter3");
    await user.click(screen.getAllByRole("button", { name: "Install git" }).at(-1)!);
    expect(await screen.findByText("that sudo password was wrong")).toBeInTheDocument();

    await user.clear(screen.getByLabelText("Sudo password"));
    await user.type(screen.getByLabelText("Sudo password"), "hunter2");
    await user.click(screen.getAllByRole("button", { name: "Install git" }).at(-1)!);
    await waitFor(() => expect(toast.success).toHaveBeenCalledWith("git installed on Home"));
    expect(mocks.post).toHaveBeenNthCalledWith(1, "/api/pairing/computers/c1/tools", { tool: "git", sudo_password: undefined });
    expect(mocks.post).toHaveBeenLastCalledWith("/api/pairing/computers/c1/tools", { tool: "git", sudo_password: "hunter2" });
    await waitFor(() => expect(screen.queryByText("Sudo password for Home")).not.toBeInTheDocument());
  });

  it("says git has no identity and sets it from the GitHub sign-in after showing it", async () => {
    serveComputers([computer({ facts: { t3: { state: "answering" }, tools: [{ name: "git", path: "/usr/bin/git" }], providers: [], projects: [] } })]);
    extra = (url) => (url === "/api/pairing/git-identity" ? { name: "Alice Example", email: "1+alice@users.noreply.github.com" } : undefined);
    mocks.post.mockResolvedValue({ data: { name: "Alice Example", email: "1+alice@users.noreply.github.com" } });
    const user = userEvent.setup();
    renderComputers(openComputer());

    await pane("Home");
    expect(screen.getByText("Not set")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Use my GitHub identity" }));
    expect(await screen.findByText("Alice Example <1+alice@users.noreply.github.com>")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Set identity" }));
    await waitFor(() => expect(mocks.post).toHaveBeenCalledWith("/api/pairing/computers/c1/git-identity"));
  });

  it("offers no identity when git already has one, or when an older app can't say whether git is there", async () => {
    const t3 = { state: "answering" };
    serveComputers([
      computer({ facts: { t3, tools: [{ name: "git", path: "/usr/bin/git" }], git_name: "Alice", git_email: "alice@example.com", providers: [], projects: [] } }),
      computer({ id: "c2", name: "Old", facts: { t3, providers: [], projects: [] } }),
    ]);
    const user = userEvent.setup();
    renderComputers(openComputer());

    await pane("Home");
    expect(screen.getByText("Alice <alice@example.com>")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Use my GitHub identity" })).not.toBeInTheDocument();
    await goTo(user, "Old");
    expect(screen.queryByRole("button", { name: "Use my GitHub identity" })).not.toBeInTheDocument();
  });

  it("asks to open a closed desktop app", async () => {
    serveComputers([computer({ server_url: "", facts: { t3: { state: "not_running", install: "desktop_app" }, providers: [], projects: [] } })]);
    renderComputers(openComputer());

    await pane("Home");
    expect(screen.getByText("Not running")).toBeInTheDocument();
    expect(screen.getByText(/Open T3 Code · desktop app/)).toBeInTheDocument();
  });

  it("says why a stopped background service could not be restarted, or when it was", async () => {
    const t3 = { state: "not_running", install: "service", restarted_at: "2026-10-10T12:00:00Z" };
    serveComputers([
      computer({ facts: { t3: { ...t3, restart_error: "the user service manager is not running" }, providers: [], projects: [] } }),
      computer({ id: "c2", name: "Desk", facts: { t3, providers: [], projects: [] } }),
    ]);
    const user = userEvent.setup();
    renderComputers(openComputer());

    await pane("Home");
    expect(screen.getByText("Restart failed: the user service manager is not running")).toBeInTheDocument();
    expect(screen.queryAllByText(/^restarted /)).toHaveLength(0);
    await goTo(user, "Desk");
    expect(screen.getAllByText(/^restarted /)).toHaveLength(1);
  });

  it("offers a computer paired before the app only its move and Remove", async () => {
    const user = userEvent.setup();
    serveComputers([pairedBefore({ token_expires_at: "0001-01-01T00:00:00Z", tunnel: { tunnel_id: "tun-1", hostname: "laptop-ab12cd34.example.com" } })]);
    renderComputers(openComputer());

    expect(await screen.findByText("Move this computer to the Nexul app")).toBeInTheDocument();
    expect(screen.getByText(/agent work can't run here until it moves/i)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /^pair$|re-pair|set up/i })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Actions for Home" }));
    expect(screen.getAllByRole("button").map((b) => b.textContent)).toContain("Remove");
    expect(screen.queryByRole("button", { name: /re-pair|set up/i })).not.toBeInTheDocument();
  });

  it("badges an unconfirmed computer and opens the dialog straight at Set up for it", async () => {
    serveComputers([computer()]);
    const user = userEvent.setup();
    renderComputers(openComputer());

    expect(await screen.findByText("Needs setup")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /^set up$/i }));
    expect(await screen.findByRole("dialog", { name: /set up home/i })).toBeInTheDocument();
    expect(await screen.findByRole("button", { name: /set up all/i })).toBeInTheDocument();
  });

  it("opens the named computer's Set up step from the setup link, and closing it forgets the link", async () => {
    serveComputers([computer(), computer({ id: "c2", name: "Mint" })]);
    const user = userEvent.setup();
    const { router } = renderComputers(`${openComputer("c2")}&setup=c2`);

    expect(await screen.findByRole("dialog", { name: /set up mint/i })).toBeInTheDocument();
    expect(screen.queryByRole("dialog", { name: /set up home/i })).not.toBeInTheDocument();
    await user.keyboard("{Escape}");
    await waitFor(() => expect(screen.queryByRole("dialog", { name: /set up/i })).not.toBeInTheDocument());
    expect(router.state.location.search).toBe("?computers=c2");
  });

  it("lists each provider with its confirmed-at time and follows a setup push without a refresh, changing nothing itself", async () => {
    const at = new Date(Date.now() - 2 * 24 * 60 * 60 * 1000).toISOString();
    setup = {
      computer_id: "c1",
      skipped_providers: [],
      models: {},
      confirmed_at: at,
      providers: [{ provider: "codex", confirmed_at: at, skills: ["tdd"], skills_version: "v1", skills_outdated: false }],
      turns: [{ run_id: "r0", turn_id: "t0", provider: "codex", provider_name: "Codex", state: "confirmed", status: "Confirmed", updated_at: at }],
    };
    serveComputers([computer()]);
    const { client } = renderComputers(openComputer());

    expect(await screen.findByText("Setup confirmed")).toBeInTheDocument();
    expect(details().getAllByTitle(/^Codex/).length).toBeGreaterThan(0);
    expect(screen.getByText("confirmed 2d ago")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Actions for Home" }));
    expect(screen.getByRole("button", { name: /re-run setup/i })).toBeInTheDocument();
    await userEvent.keyboard("{Escape}");

    setup = {
      ...setup,
      turns: [{ run_id: "r1", turn_id: "t1", provider: "codex", provider_name: "Codex", state: "running", status: "Installing", updated_at: at }],
    };
    await act(() => client.invalidateQueries({ queryKey: ["getComputerSetup"] }));
    expect(await screen.findByText("setting up…")).toBeInTheDocument();
    expect(mocks.post).not.toHaveBeenCalled();
    expect(mocks.put).not.toHaveBeenCalled();
    expect(mocks.del).not.toHaveBeenCalled();
  });

  it("leads with Update skills when out-of-date skills are all that is left, and starts it in the Set up dialog", async () => {
    const at = new Date(Date.now() - 2 * 24 * 60 * 60 * 1000).toISOString();
    setup = {
      computer_id: "c1",
      confirmed_at: at,
      providers: [
        { provider: "codex", confirmed_at: at, skills: ["tdd"], skills_version: "old", skills_outdated: true },
        { provider: "opencode", confirmed_at: at, skills: ["tdd"], skills_version: "old", skills_outdated: true },
      ],
      skipped_providers: [],
      models: {},
      model_options: {},
      folder: "",
      turns: [],
    };
    serveComputers([computer()]);
    mocks.post.mockResolvedValue({ data: { run_id: "r1", computer_id: "c1", providers: [{ provider: "codex", name: "Codex" }] } });
    const user = userEvent.setup();
    renderComputers(openComputer());

    expect(await screen.findAllByText("skills out of date")).toHaveLength(2);
    expect(screen.getByText("Skills out of date")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Actions for Home" }));
    expect(screen.getByRole("button", { name: /re-run setup/i })).toBeInTheDocument();
    await user.keyboard("{Escape}");
    await user.click(screen.getByRole("button", { name: /update skills/i }));

    await waitFor(() => expect(mocks.post).toHaveBeenCalledWith("/api/pairing/computers/c1/setup/skills"));
    expect(mocks.post).toHaveBeenCalledTimes(1);
    expect(await screen.findByRole("dialog", { name: /set up home/i })).toBeInTheDocument();
  });

  it("keeps Re-run setup alone when something besides skills needs setup", async () => {
    const at = new Date(Date.now() - 2 * 24 * 60 * 60 * 1000).toISOString();
    setup = {
      computer_id: "c1",
      skipped_providers: [],
      models: {},
      confirmed_at: at,
      providers: [{ provider: "codex", confirmed_at: at, skills: ["tdd"], skills_version: "old", skills_outdated: true }],
      turns: [{ run_id: "r0", turn_id: "t0", provider: "opencode", provider_name: "OpenCode", kind: "setup", state: "failed", status: "npx missing", updated_at: at }],
    };
    serveComputers([computer()]);
    renderComputers(openComputer());

    expect(await screen.findByText("setup failed")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /re-run setup/i })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /update skills/i })).not.toBeInTheDocument();
  });

  describe("a computer added with its app", () => {
    const relay = "http://c1.nexul-computer.invalid";
    const answering = { t3: { state: "answering", install: "service", port: 3773 }, providers: [], projects: [] };
    const online = { connected: true, last_seen: "2026-10-10T12:00:00Z", reconnecting: false };

    it.each([
      ["online and answering", { runner: online, facts: answering }, "connected", ["Online", "T3 Code answering"]],
      ["online with its desktop app closed", { runner: online, facts: { ...answering, t3: { state: "not_running", install: "desktop_app" } } }, undefined, ["Online", "Open T3 Code"]],
      ["online but refused pairing", { runner: online, facts: answering, pair_error: "T3 Code made no pairing token" }, undefined, ["Online", "Pairing failed"]],
      ["online with its session refused by T3 Code", { runner: online, facts: answering }, "rejected", ["Online", "T3 Code refused the session"]],
      ["back any moment after a drop", { runner: { connected: false, last_seen: "2026-10-10T12:00:00Z", reconnecting: true } }, undefined, ["Reconnecting"]],
      ["offline", { runner: { connected: false, last_seen: "2026-10-10T12:00:00Z" } }, undefined, ["Offline"]],
    ])("reads each lane on its own when %s", async (_, state, lane, lanes) => {
      serveComputers([computer({ server_url: relay, ...state })]);
      presence = lane ? { c1: lane } : {};
      renderComputers(openComputer());

      await pane("Home");
      for (const text of lanes) expect(await details().findByText(text)).toBeInTheDocument();
      expect(screen.queryByRole("button", { name: /add this computer again/i })).not.toBeInTheDocument();
    });

    it("doesn't call T3 Code running from the last report of a computer that went offline", async () => {
      const facts = { ...answering, t3: { ...answering.t3, version: "0.1.1", install: "command_line" } };
      serveComputers([computer({ server_url: relay, runner: { connected: false, last_seen: "2026-10-10T12:00:00Z" }, facts })]);
      renderComputers(openComputer());

      await pane("Home");
      expect(screen.getByText("Unknown while offline")).toBeInTheDocument();
      expect(screen.getByText("port 3773 · 0.1.1 · command line")).toBeInTheDocument();
      expect(screen.queryByText("Running")).not.toBeInTheDocument();
    });

    it("offers Add this computer again once its app is gone, with a fresh command for the same computer", async () => {
      serveComputers([computer({ server_url: relay, runner: undefined, token_expires_at: "0001-01-01T00:00:00Z" })]);
      const command = "curl -fsSL https://nexul.io/computer.sh | sudo sh -s -- eyJ.fresh";
      mocks.post.mockResolvedValue({ data: { computer: computer({ server_url: relay }), token: "eyJ.fresh", expires_at: "2026-10-10T13:00:00Z", commands: { unix: command, windows: "" } } });
      const user = userEvent.setup();
      renderComputers(openComputer());

      expect(await details().findByText("Nexul app not installed")).toBeInTheDocument();
      await user.click(screen.getByRole("button", { name: /add this computer again/i }));
      expect(await screen.findByRole("heading", { name: "Add Home again" })).toBeInTheDocument();
      expect(await screen.findByText(command)).toBeInTheDocument();
      expect(mocks.post).toHaveBeenCalledWith("/api/pairing/computers/enrollments", { id: "c1" });
    });

    it("re-pairs now through its app instead of asking for a link, saying so in the pane until it is done", async () => {
      serveComputers([computer({ server_url: relay, runner: online, facts: answering })]);
      let finish: (value: unknown) => void = () => {};
      mocks.post.mockReturnValue(new Promise((resolve) => (finish = resolve)));
      const user = userEvent.setup();
      renderComputers(openComputer());

      await user.click(await screen.findByRole("button", { name: "Actions for Home" }));
      await user.click(screen.getByRole("button", { name: "Re-pair now" }));
      await waitFor(() => expect(mocks.post).toHaveBeenCalledWith("/api/pairing/computers/c1/pair"));
      expect(await details().findByText("Re-pairing")).toBeInTheDocument();
      expect(screen.queryByLabelText(/pairing link/i)).not.toBeInTheDocument();

      await act(async () => finish({ data: computer({ server_url: relay }) }));
      await waitFor(() => expect(screen.queryByText("Re-pairing")).not.toBeInTheDocument());
      expect(toast.success).toHaveBeenCalledWith("Home re-paired");
    });

    it("renames it from its menu", async () => {
      serveComputers([computer({ server_url: relay, runner: online })]);
      mocks.patch.mockResolvedValue({ data: computer({ name: "Work laptop", server_url: relay }) });
      const user = userEvent.setup();
      renderComputers(openComputer());

      await user.click(await screen.findByRole("button", { name: "Actions for Home" }));
      await user.click(screen.getByRole("button", { name: "Rename" }));
      const name = await screen.findByLabelText(/^name$/i);
      await user.clear(name);
      await user.type(name, "Work laptop");
      await user.click(screen.getByRole("button", { name: /^rename$/i }));
      await waitFor(() => expect(mocks.patch).toHaveBeenCalledWith("/api/pairing/computers/c1", { name: "Work laptop" }));
    });
  });

  it("restarts the Nexul app on an idle computer from its menu at once", async () => {
    serveComputers([computer({ running: 0 })]);
    mocks.post.mockResolvedValue({ data: undefined });
    const user = userEvent.setup();
    renderComputers(openComputer());

    await user.click(await screen.findByRole("button", { name: "Actions for Home" }));
    await user.click(screen.getByRole("button", { name: "Restart Nexul app" }));

    await waitFor(() => expect(mocks.post).toHaveBeenCalledWith("/api/pairing/computers/c1/restart"));
    expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument();
    expect(toast.success).toHaveBeenCalledWith("Restarting the Nexul app on Home");
  });

  it("asks before restarting the Nexul app while play runs are going on the computer", async () => {
    serveComputers([computer({ running: 2 })]);
    mocks.post.mockResolvedValue({ data: undefined });
    const user = userEvent.setup();
    renderComputers(openComputer());

    await user.click(await screen.findByRole("button", { name: "Actions for Home" }));
    await user.click(screen.getByRole("button", { name: "Restart Nexul app" }));
    expect(await screen.findByText(/2 play runs are going on it/)).toBeInTheDocument();
    expect(mocks.post).not.toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: "Restart Nexul app" }));
    await waitFor(() => expect(mocks.post).toHaveBeenCalledWith("/api/pairing/computers/c1/restart"));
  });

  it("offers no restart for an offline computer", async () => {
    serveComputers([computer({ running: 0, runner: { connected: false, last_seen: "2026-10-10T12:00:00Z", reconnecting: false } })]);
    const user = userEvent.setup();
    renderComputers(openComputer());

    await user.click(await screen.findByRole("button", { name: "Actions for Home" }));
    expect(screen.queryByRole("button", { name: "Restart Nexul app" })).not.toBeInTheDocument();
  });

  it("removes a computer from its menu only after the confirm", async () => {
    serveComputers([computer()]);
    mocks.del.mockResolvedValue({ data: {} });
    const user = userEvent.setup();
    renderComputers(openComputer());

    await user.click(await screen.findByRole("button", { name: "Actions for Home" }));
    await user.click(screen.getByRole("button", { name: /^remove$/i }));
    expect(mocks.del).not.toHaveBeenCalled();

    await user.click(await screen.findByRole("button", { name: /^remove computer$/i }));
    await waitFor(() => expect(mocks.del).toHaveBeenCalledWith("/api/pairing/computers/c1"));
    await waitFor(() =>
      expect(toast.success).toHaveBeenLastCalledWith(
        "Computer removed",
        expect.objectContaining({ description: expect.stringMatching(/session on this computer stays valid until .*t3 auth session revoke/) }),
      ),
    );
  });

  it("removes a computer still pairing without a leftover-session note", async () => {
    serveComputers([computer({ token_expires_at: "0001-01-01T00:00:00Z" })]);
    mocks.del.mockResolvedValue({ data: {} });
    const user = userEvent.setup();
    renderComputers(openComputer());

    await user.click(await screen.findByRole("button", { name: "Actions for Home" }));
    await user.click(screen.getByRole("button", { name: /^remove$/i }));
    await user.click(await screen.findByRole("button", { name: /^remove computer$/i }));
    await waitFor(() => expect(toast.success).toHaveBeenLastCalledWith("Computer removed", undefined));
  });
});
