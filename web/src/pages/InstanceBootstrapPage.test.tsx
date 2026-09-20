import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { InstanceBootstrapPage } from "@/pages/InstanceBootstrapPage";

const mocks = vi.hoisted(() => ({
  post: vi.fn(),
  errorMessage: vi.fn(),
  toast: { success: vi.fn(), error: vi.fn() },
  assign: vi.fn(),
}));

vi.mock("@/api/client", () => ({
  api: { post: mocks.post },
  errorMessage: mocks.errorMessage,
  joinAPIURL: (path: string) => `http://localhost:8080${path}`,
}));

vi.mock("sonner", () => ({ toast: mocks.toast }));

const VERIFY_URL = "/api/auth/bootstrap/verify";

// Beat one: Verify ticks both rows; the submit button only becomes "Set up instance" once they are green.
const verifyAndPass = async (user: ReturnType<typeof userEvent.setup>) => {
  await user.click(screen.getByRole("button", { name: /^verify$/i }));
  await screen.findByRole("button", { name: /set up instance/i });
};

// The step indicator is a status role too, so only count rows inside the checks list.
const checkRows = () => within(screen.getByRole("list", { name: /github app checks/i })).getAllByRole("status");

const renderPage = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <InstanceBootstrapPage />
    </QueryClientProvider>,
  );
};

describe("InstanceBootstrapPage", () => {
  beforeEach(() => {
    mocks.post.mockReset();
    mocks.errorMessage.mockClear();
    mocks.toast.error.mockClear();
    mocks.assign.mockClear();
    vi.stubGlobal("location", { ...window.location, assign: mocks.assign });
  });

  it("renders the instance url, client id, client secret, app slug fields and the derived callback preview", async () => {
    const user = userEvent.setup();
    renderPage();

    expect(screen.getByLabelText(/instance url/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/github oauth client id/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/github app slug/i)).toBeInTheDocument();
    const secretInput = screen.getByLabelText(/github oauth client secret/i);
    expect(secretInput).toHaveAttribute("type", "password");

    const input = screen.getByLabelText(/instance url/i);
    await user.clear(input);
    await user.type(input, "https://deploy.example.com");
    expect(screen.getByText(/https:\/\/deploy\.example\.com\/auth\/callback/i)).toBeInTheDocument();
  });

  it("verifies the instance url, slug and secret as three rows before it will set up the instance", async () => {
    mocks.post.mockResolvedValue({ data: { configured: true } });
    const user = userEvent.setup();
    renderPage();

    const rows = checkRows();
    expect(rows).toHaveLength(3);
    rows.forEach((row) => expect(row).toHaveAttribute("data-state", "idle"));
    expect(screen.queryByRole("button", { name: /set up instance/i })).not.toBeInTheDocument();

    const urlInput = screen.getByLabelText(/instance url/i);
    await user.clear(urlInput);
    await user.type(urlInput, "https://deploy.example.com");
    await user.type(screen.getByLabelText(/github oauth client id/i), "client-id-123");
    await user.type(screen.getByLabelText(/github oauth client secret/i), "client-secret-456");
    await user.type(screen.getByLabelText(/github app slug/i), "my-app");
    await verifyAndPass(user);

    const fields = {
      instance_url: "https://deploy.example.com",
      client_id: "client-id-123",
      client_secret: "client-secret-456",
      app_slug: "my-app",
    };
    expect(mocks.post).toHaveBeenCalledWith(VERIFY_URL, fields, { params: { check: "instance_url" } });
    expect(mocks.post).toHaveBeenCalledWith(VERIFY_URL, fields, { params: { check: "slug" } });
    expect(mocks.post).toHaveBeenCalledWith(VERIFY_URL, fields, { params: { check: "secret" } });
    expect(mocks.post).not.toHaveBeenCalledWith("/api/auth/bootstrap", expect.anything());
    checkRows().forEach((row) => expect(row).toHaveAttribute("data-state", "ok"));
  });

  it("marks only the failed check red and keeps Verify available", async () => {
    mocks.post.mockImplementation((_url: string, _body: unknown, config?: { params?: { check?: string } }) =>
      config?.params?.check === "slug"
        ? Promise.reject({ response: { status: 400, data: { message: "no GitHub App with slug" } } })
        : Promise.resolve({}),
    );
    mocks.errorMessage.mockReturnValue("no GitHub App with slug");
    const user = userEvent.setup();
    renderPage();

    await user.type(screen.getByLabelText(/github oauth client id/i), "client-id-123");
    await user.type(screen.getByLabelText(/github oauth client secret/i), "client-secret-456");
    await user.type(screen.getByLabelText(/github app slug/i), "nope");
    await user.click(screen.getByRole("button", { name: /^verify$/i }));

    const rows = checkRows();
    await vi.waitFor(() => expect(rows[1]).toHaveAttribute("data-state", "failed"));
    expect(rows[1]).toHaveTextContent(/no GitHub App with slug/i);
    expect(rows[0]).toHaveAttribute("data-state", "ok");
    expect(rows[2]).toHaveAttribute("data-state", "ok");
    expect(screen.getByRole("button", { name: /^verify$/i })).toBeEnabled();
    expect(mocks.assign).not.toHaveBeenCalled();
  });

  it("puts the rows back to idle when a verified field changes", async () => {
    mocks.post.mockResolvedValue({});
    const user = userEvent.setup();
    renderPage();

    await user.type(screen.getByLabelText(/github oauth client id/i), "client-id-123");
    await user.type(screen.getByLabelText(/github oauth client secret/i), "client-secret-456");
    await user.type(screen.getByLabelText(/github app slug/i), "my-app");
    await verifyAndPass(user);

    await user.type(screen.getByLabelText(/github app slug/i), "x");
    checkRows().forEach((row) => expect(row).toHaveAttribute("data-state", "idle"));
    expect(screen.getByRole("button", { name: /^verify$/i })).toBeInTheDocument();
  });

  it("submits the form and redirects the browser to /auth/github on success", async () => {
    mocks.post.mockResolvedValue({ data: { configured: true } });
    const user = userEvent.setup();
    renderPage();

    const urlInput = screen.getByLabelText(/instance url/i);
    await user.clear(urlInput);
    await user.type(urlInput, "https://deploy.example.com");
    await user.type(screen.getByLabelText(/github oauth client id/i), "client-id-123");
    await user.type(screen.getByLabelText(/github oauth client secret/i), "client-secret-456");
    await user.type(screen.getByLabelText(/github app slug/i), "my-app");
    await verifyAndPass(user);
    await user.click(screen.getByRole("button", { name: /set up instance/i }));

    expect(mocks.post).toHaveBeenCalledWith("/api/auth/bootstrap", {
      instance_url: "https://deploy.example.com",
      client_id: "client-id-123",
      client_secret: "client-secret-456",
      app_slug: "my-app",
    });
    expect(mocks.assign).toHaveBeenCalledWith("http://localhost:8080/auth/github");
  });

  it("shows GitHub's rejection inline and keeps the form open", async () => {
    // Verify passes, then the store step itself is refused: the verdict still lands next to the fields.
    mocks.post.mockImplementation((url: string) =>
      url === VERIFY_URL
        ? Promise.resolve({})
        : Promise.reject({ response: { status: 400, data: { message: "invalid: GitHub rejected the client secret" } } }),
    );
    mocks.errorMessage.mockReturnValue("invalid: GitHub rejected the client secret");
    const user = userEvent.setup();
    renderPage();

    await user.type(screen.getByLabelText(/github oauth client id/i), "client-id-123");
    await user.type(screen.getByLabelText(/github oauth client secret/i), "wrong");
    await user.type(screen.getByLabelText(/github app slug/i), "my-app");
    await verifyAndPass(user);
    await user.click(screen.getByRole("button", { name: /set up instance/i }));

    expect(await screen.findByRole("alert")).toHaveTextContent(/rejected the client secret/i);
    expect(mocks.assign).not.toHaveBeenCalled();
  });

  it("handles a 409 conflict response gracefully instead of crashing", async () => {
    mocks.post.mockImplementation((url: string) =>
      url === VERIFY_URL
        ? Promise.resolve({})
        : Promise.reject({ response: { status: 409, data: { message: "already configured" } } }),
    );
    mocks.errorMessage.mockReturnValue("already configured");
    const user = userEvent.setup();
    renderPage();

    await user.type(screen.getByLabelText(/github oauth client id/i), "client-id-123");
    await user.type(screen.getByLabelText(/github oauth client secret/i), "client-secret-456");
    await user.type(screen.getByLabelText(/github app slug/i), "my-app");
    await verifyAndPass(user);
    await user.click(screen.getByRole("button", { name: /set up instance/i }));

    expect(await screen.findByText(/already configured/i)).toBeInTheDocument();
    expect(mocks.assign).not.toHaveBeenCalled();
  });
});
