import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useForm, useWatch } from "react-hook-form";
import { describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { HarnessProjectField } from "@/components/settings/HarnessProjectField";

const Harness = ({ computerId, saved = "" }: { computerId: string; saved?: string }) => {
  const form = useForm<{ fallback_project_id: string }>({ defaultValues: { fallback_project_id: saved } });
  return <HarnessProjectField control={form.control} name="fallback_project_id" label="Fallback T3 project" computerId={computerId} />;
};

const renderField = (computerId: string, saved = "") => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <Harness computerId={computerId} saved={saved} />
    </QueryClientProvider>,
  );
};

describe("HarnessProjectField", () => {
  it("renders a picker from the computer's registry", async () => {
    vi.spyOn(api, "get").mockResolvedValueOnce({
      data: { projects: [{ id: "proj-1", title: "My App" }] },
    } as never);
    renderField("comp-1");
    expect(await screen.findByText("Fallback T3 project")).toBeInTheDocument();
    expect(await screen.findByRole("combobox")).toBeInTheDocument();
  });

  it("falls back to manual entry when the computer is unreachable", async () => {
    vi.spyOn(api, "get").mockRejectedValueOnce(new Error("down"));
    renderField("comp-1");
    expect(await screen.findByText(/paste the project id/)).toBeInTheDocument();
    expect(screen.getByPlaceholderText("e.g. proj_abc123")).toBeInTheDocument();
  });

  it("hides a saved id while the computer is unreachable, and Change clears it for a new one", async () => {
    vi.spyOn(api, "get").mockRejectedValueOnce(new Error("down"));
    renderField("comp-1", "proj-1");
    expect(await screen.findByText(/back online/)).toBeInTheDocument();
    expect(screen.queryByDisplayValue("proj-1")).not.toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: "Change" }));

    expect(screen.queryByDisplayValue("proj-1")).not.toBeInTheDocument();
    expect(screen.getByPlaceholderText("e.g. proj_abc123")).toBeInTheDocument();
  });

  it("shows only a loader while the registry loads, never the saved id", () => {
    vi.spyOn(api, "get").mockReturnValueOnce(new Promise(() => {}));
    renderField("comp-1", "proj-1");
    expect(screen.getByText("Loading projects")).toBeInTheDocument();
    expect(screen.queryByDisplayValue("proj-1")).not.toBeInTheDocument();
  });

  it("prompts for a computer before loading anything", () => {
    renderField("");
    expect(screen.getByText(/Pick a computer/)).toBeInTheDocument();
    expect(screen.getByPlaceholderText("e.g. proj_abc123")).toBeInTheDocument();
  });

  it("clones a project repository the computer lacks and picks it", async () => {
    const onComputer = [{ id: "t3-web", title: "web", path: "/home/me/Code/web" }];
    vi.spyOn(api, "get").mockImplementation(async (url: string) => {
      if (url === "/api/projects/p1/repos") {
        return { data: [
          { owner: "acme", name: "web", full_name: "acme/web", connector_id: "github", role: "app" },
          { owner: "acme", name: "api", full_name: "acme/api", connector_id: "github", role: "app" },
        ] } as never;
      }
      return { data: { projects: [...onComputer], can_clone: true } } as never;
    });
    const post = vi.spyOn(api, "post").mockImplementation(async () => {
      onComputer.push({ id: "t3-api", title: "api", path: "/home/me/Code/api" });
      return { data: onComputer[1] } as never;
    });
    const Picked = () => {
      const form = useForm<{ harness_project_id: string }>({ defaultValues: { harness_project_id: "" } });
      const picked = useWatch({ control: form.control, name: "harness_project_id" });
      return (
        <>
          <HarnessProjectField control={form.control} name="harness_project_id" label="T3 project" computerId="comp-1" projectId="p1" />
          <output>picked:{picked}</output>
        </>
      );
    };
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <Picked />
      </QueryClientProvider>,
    );

    expect(screen.queryByRole("button", { name: "Create empty project" })).not.toBeInTheDocument();
    await userEvent.click(await screen.findByRole("button", { name: "Clone acme/api" }));

    expect(screen.queryByRole("button", { name: "Clone acme/web" })).not.toBeInTheDocument();
    expect(post).toHaveBeenCalledWith("/api/pairing/computers/comp-1/projects", { repository: "acme/api" });
    expect(await screen.findByText("picked:t3-api")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Clone/ })).not.toBeInTheDocument();
  });

  it("says to pair again before a click when the computer's session may not clone", async () => {
    vi.spyOn(api, "get").mockImplementation(async (url: string) => {
      if (url === "/api/projects/p1/repos") {
        return { data: [{ owner: "acme", name: "api", full_name: "acme/api", connector_id: "github", role: "app" }] } as never;
      }
      if (url === "/api/pairing/computers") {
        return { data: { computers: [{ id: "comp-1", name: "Studio" }] } } as never;
      }
      return { data: { projects: [], can_clone: false } } as never;
    });
    const post = vi.spyOn(api, "post");
    const Field = () => {
      const form = useForm<{ harness_project_id: string }>({ defaultValues: { harness_project_id: "" } });
      return <HarnessProjectField control={form.control} name="harness_project_id" label="T3 project" computerId="comp-1" projectId="p1" />;
    };
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <Field />
      </QueryClientProvider>,
    );

    expect(await screen.findByText(/paired before Nexul could clone there/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Pair again" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Clone acme/api" })).toBeDisabled();
    expect(post).not.toHaveBeenCalled();
  });

  it("offers an empty project when there is nothing to clone, and picks the one it makes", async () => {
    vi.spyOn(api, "get").mockImplementation(async (url: string) => {
      if (url === "/api/projects/p1/repos") return { data: [] } as never;
      return { data: { projects: [], can_clone: true } } as never;
    });
    const post = vi.spyOn(api, "post").mockResolvedValue({ data: { id: "t3-notes", title: "notes", path: "~/Code/notes" } } as never);
    const Picked = () => {
      const form = useForm<{ harness_project_id: string }>({ defaultValues: { harness_project_id: "" } });
      const picked = useWatch({ control: form.control, name: "harness_project_id" });
      return (
        <>
          <HarnessProjectField control={form.control} name="harness_project_id" label="T3 project" computerId="comp-1" projectId="p1" />
          <output>picked:{picked}</output>
        </>
      );
    };
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <Picked />
      </QueryClientProvider>,
    );

    const create = await screen.findByRole("button", { name: "Create empty project" });
    expect(create).toBeDisabled();
    await userEvent.type(screen.getByRole("textbox", { name: "Folder name" }), " notes ");
    await userEvent.click(create);

    expect(post).toHaveBeenCalledWith("/api/pairing/computers/comp-1/projects", { name: "notes" });
    expect(await screen.findByText("picked:t3-notes")).toBeInTheDocument();
  });

  it("holds the empty project back until the project's repositories are known", async () => {
    vi.spyOn(api, "get").mockImplementation(async (url: string) => {
      if (url === "/api/projects/p1/repos") return new Promise(() => {});
      return { data: { projects: [], can_clone: true } } as never;
    });
    const Field = () => {
      const form = useForm<{ harness_project_id: string }>({ defaultValues: { harness_project_id: "" } });
      return <HarnessProjectField control={form.control} name="harness_project_id" label="T3 project" computerId="comp-1" projectId="p1" />;
    };
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <Field />
      </QueryClientProvider>,
    );

    expect(await screen.findByText(/has no T3 projects yet/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Create empty project" })).not.toBeInTheDocument();
  });
});
