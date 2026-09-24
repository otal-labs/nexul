import { describe, expect, it } from "vitest";

import { ReviewStatus } from "@/models/CodeReview";
import { InstanceUpgradeSchema, isUpgradeInProgress, UpgradeRecordStatus } from "@/models/InstanceUpgrade";
import { NotificationKind, SubjectType } from "@/models/Notification";
import { SaveDocFormSchema } from "@/models/Doc";
import { QueuedJobSchema, RequestKind, RunnerSchema } from "@/models/Runner";
import { cardPerson, reporterLabel, SaveTicketFormSchema, TicketStatus, type Ticket } from "@/models/Ticket";
import { VersionSchema } from "@/models/Version";

describe("ReviewStatus", () => {
  it("mirrors the backend review status strings", () => {
    expect(ReviewStatus.Pending).toBe("pending");
    expect(ReviewStatus.Approved).toBe("approved");
    expect(ReviewStatus.ChangesRequested).toBe("changes_requested");
    expect(ReviewStatus.Merged).toBe("merged");
  });
});

describe("RunnerSchema", () => {
  it("parses an online runner with a running job", () => {
    const data = RunnerSchema.parse({
      id: "r-1",
      name: "alpha",
      connected: true,
      last_seen: "2026-08-12T12:00:00Z",
      running_job: { id: "d-9", kind: "deploy", service: "api" },
      version: "v0.1.4",
    });
    expect(data.id).toBe("r-1");
    expect(data.running_job?.service).toBe("api");
    expect(data.version).toBe("v0.1.4");
  });

  it("parses an idle runner with a null running job", () => {
    const data = RunnerSchema.parse({
      id: "r-2",
      name: "beta",
      connected: false,
      last_seen: "2026-08-12T12:00:00Z",
      running_job: null,
      version: "",
    });
    expect(data.running_job).toBeNull();
  });

  it("rejects a runner missing its id", () => {
    expect(
      RunnerSchema.safeParse({
        id: "",
        name: "beta",
        connected: false,
        last_seen: "2026-08-12T12:00:00Z",
        running_job: null,
        version: "",
      }).success,
    ).toBe(false);
  });
});

describe("VersionSchema", () => {
  it("parses a version response with an available update", () => {
    const data = VersionSchema.parse({
      version: "v0.2.0-beta-310",
      channel: "beta",
      latest: { version: "v0.2.0-beta-331", url: "https://github.com/otal-labs/nexul/releases/tag/v0.2.0-beta-331" },
      update_available: true,
    });
    expect(data.channel).toBe("beta");
    expect(data.latest?.version).toBe("v0.2.0-beta-331");
  });

  it("parses a version response with no known update", () => {
    const data = VersionSchema.parse({
      version: "dev",
      channel: "dev",
      latest: null,
      update_available: false,
    });
    expect(data.latest).toBeNull();
  });
});

describe("InstanceUpgradeSchema", () => {
  it("parses a response with no upgrade record", () => {
    const data = InstanceUpgradeSchema.parse({
      version: "v0.2.0-beta-003",
      channel: "beta",
      latest: { version: "v0.2.0-beta-004", url: "https://github.com/otal-labs/nexul/releases/tag/v0.2.0-beta-004" },
      update_available: true,
      can_upgrade: true,
      reason: "",
      upgrade: null,
    });
    expect(data.can_upgrade).toBe(true);
    expect(data.upgrade).toBeNull();
  });

  it("parses a response with a pending upgrade record", () => {
    const data = InstanceUpgradeSchema.parse({
      version: "v0.2.0-beta-003",
      channel: "beta",
      latest: null,
      update_available: false,
      can_upgrade: false,
      reason: "an upgrade is already in progress",
      upgrade: {
        id: "u-1",
        from_version: "v0.2.0-beta-003",
        to_version: "v0.2.0-beta-004",
        status: "pending",
        error: "",
        requested_by: "user-1",
        created_at: "2026-09-15T00:00:00Z",
        updated_at: "2026-09-15T00:00:00Z",
      },
    });
    expect(data.upgrade?.status).toBe("pending");
  });
});

describe("UpgradeRecordStatus", () => {
  it("mirrors the backend upgrade status strings", () => {
    expect(UpgradeRecordStatus.Pending).toBe("pending");
    expect(UpgradeRecordStatus.Started).toBe("started");
    expect(UpgradeRecordStatus.Completed).toBe("completed");
    expect(UpgradeRecordStatus.Failed).toBe("failed");
  });
});

describe("isUpgradeInProgress", () => {
  it("is false for a null record", () => {
    expect(isUpgradeInProgress(null)).toBe(false);
  });

  it("is true for pending and started records", () => {
    const base = {
      id: "u-1",
      from_version: "v1",
      to_version: "v2",
      error: "",
      requested_by: "user-1",
      created_at: "2026-09-15T00:00:00Z",
      updated_at: "2026-09-15T00:00:00Z",
    };
    expect(isUpgradeInProgress({ ...base, status: "pending" })).toBe(true);
    expect(isUpgradeInProgress({ ...base, status: "started" })).toBe(true);
    expect(isUpgradeInProgress({ ...base, status: "completed" })).toBe(false);
    expect(isUpgradeInProgress({ ...base, status: "failed" })).toBe(false);
  });
});

describe("QueuedJobSchema", () => {
  it("parses a queued deploy", () => {
    const data = QueuedJobSchema.parse({ id: "d-1", kind: "deploy", service: "api" });
    expect(data.kind).toBe("deploy");
    expect(data.service).toBe("api");
  });

  it("rejects a queued job without an id", () => {
    expect(QueuedJobSchema.safeParse({ kind: "deploy" }).success).toBe(false);
  });
});

describe("RequestKind", () => {
  it("mirrors the backend request kinds", () => {
    expect(RequestKind.Build).toBe("build");
    expect(RequestKind.Deploy).toBe("deploy");
  });
});

describe("SaveDocFormSchema", () => {
  it("accepts project_id, title, and body", () => {
    const data = SaveDocFormSchema.parse({ project_id: "p-1", title: "Spec", body: "body" });
    expect(data.project_id).toBe("p-1");
    expect(data.title).toBe("Spec");
    expect(data.body).toBe("body");
  });

  it("rejects an empty title", () => {
    const result = SaveDocFormSchema.safeParse({ project_id: "p-1", title: "", body: "" });
    expect(result.success).toBe(false);
  });

  it("rejects an empty project_id", () => {
    const result = SaveDocFormSchema.safeParse({ project_id: "", title: "Spec", body: "" });
    expect(result.success).toBe(false);
  });
});

describe("ticket people", () => {
  it("labels each reporter kind", () => {
    expect(reporterLabel({ kind: "user", login: "onik97" })).toBe("onik97");
    expect(reporterLabel({ kind: "user:mcp", login: "onik97" })).toBe("Nexul · for onik97");
    expect(reporterLabel({ kind: "automation", automation_name: "Triage" })).toBe("Nexul · Triage");
    expect(reporterLabel({ kind: "user:mcp" })).toBe("Nexul");
  });

  it("puts the tester on the card only in a testing stage", () => {
    const t = { developer: "dev", tester: "qa" } as Ticket;
    expect(cardPerson(t, "testing")).toEqual({ role: "tester", login: "qa" });
    expect(cardPerson(t, "review")).toEqual({ role: "developer", login: "dev" });
    expect(cardPerson(t, undefined)).toEqual({ role: "developer", login: "dev" });
  });
});

describe("SaveTicketFormSchema", () => {
  it("accepts a ticket with optional fields", () => {
    const data = SaveTicketFormSchema.parse({ title: "Fix", body: "b", project_id: "p-1", doc_id: "doc-1", developer: "onik97", tester: "lena" });
    expect(data.project_id).toBe("p-1");
    expect(data.doc_id).toBe("doc-1");
    expect(data.developer).toBe("onik97");
    expect(data.tester).toBe("lena");
  });

  it("requires a project", () => {
    expect(SaveTicketFormSchema.safeParse({ title: "Fix", body: "" }).success).toBe(false);
  });

  it("rejects an empty title", () => {
    expect(SaveTicketFormSchema.safeParse({ title: " ", project_id: "p-1" }).success).toBe(false);
  });

  it("mirrors the backend status strings", () => {
    expect(TicketStatus.Open).toBe("open");
    expect(TicketStatus.InProgress).toBe("in_progress");
    expect(TicketStatus.Done).toBe("done");
    expect(TicketStatus.Closed).toBe("closed");
  });
});

import { LinkBranchFormSchema, LinkPRFormSchema, PRLinkState } from "@/models/Ticket";
import { DeployStatus, DeployStrategy, ServiceFormSchema } from "@/models/Service";
import { parseEnv, formatEnv } from "@/lib/env";

describe("Service models", () => {
  it("mirrors the backend strategy and status strings", () => {
    expect(DeployStrategy.Compose).toBe("compose");
    expect(DeployStrategy.Run).toBe("run");
    expect(DeployStatus.Pending).toBe("pending");
    expect(DeployStatus.Healthy).toBe("healthy");
    expect(DeployStatus.Failed).toBe("failed");
  });

  it("validates the create form per strategy", () => {
    const compose = ServiceFormSchema.parse({
      name: "api",
      target: "10.0.0.1:22",
      strategy: "compose",
      compose_dir: "/srv/api",
      docker_network: "",
      health_url: "http://10.0.0.1:8080/health",
      env: "PORT=8080",
      image: "",
      ref: "",
    });
    expect(compose.compose_dir).toBe("/srv/api");

    expect(
      ServiceFormSchema.safeParse({
        name: "api",
        target: "h:22",
        strategy: "run",
        compose_dir: "",
        docker_network: "",
        health_url: "http://h/health",
        env: "",
        image: "",
        ref: "",
      }).success,
    ).toBe(false);

    expect(
      ServiceFormSchema.safeParse({
        name: "api",
        target: "h:22",
        strategy: "compose",
        compose_dir: "",
        docker_network: "",
        health_url: "http://h/health",
        env: "",
        image: "",
        ref: "",
      }).success,
    ).toBe(false);
  });

  it("requires a name, target, and health URL", () => {
    expect(
      ServiceFormSchema.safeParse({
        name: "",
        target: "h:22",
        strategy: "run",
        compose_dir: "",
        docker_network: "net",
        health_url: "http://h/health",
        env: "",
        image: "",
        ref: "",
      }).success,
    ).toBe(false);
  });

  it("parses and formats env blocks", () => {
    expect(parseEnv("PORT=8080\n# comment\nEMPTY=\nNEXT=2")).toEqual({ PORT: "8080", EMPTY: "", NEXT: "2" });
    expect(formatEnv({ A: "1", B: "2" })).toBe("A=1\nB=2");
  });
});

describe("PR link forms", () => {
  it("mirrors the PR link state strings", () => {
    expect(PRLinkState.Merged).toBe("merged");
  });

  it("validates link forms", () => {
    expect(() => LinkPRFormSchema.parse({ owner: "acme", repo: "app", number: 0 })).toThrow();
    expect(() => LinkBranchFormSchema.parse({ owner: "", repo: "app", branch: "x" })).toThrow();
    expect(
      LinkBranchFormSchema.parse({ owner: "acme", repo: "app", branch: "ticket/1" }).branch,
    ).toBe("ticket/1");
  });
});

describe("NotificationKind", () => {
  it("mirrors the backend notification kind strings", () => {
    expect(NotificationKind.TicketAssigned).toBe("ticket.assigned");
    expect(NotificationKind.TicketMentioned).toBe("ticket.mentioned");
    expect(NotificationKind.TicketStatusChanged).toBe("ticket.status_changed");
    expect(NotificationKind.DocCreated).toBe("doc.created");
    expect(NotificationKind.DocUpdated).toBe("doc.updated");
  });
});

describe("SubjectType", () => {
  it("mirrors the backend subject type strings", () => {
    expect(SubjectType.Ticket).toBe("ticket");
    expect(SubjectType.Doc).toBe("doc");
  });
});
