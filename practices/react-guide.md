# React practices

How the web client in `web/` is written and reviewed. The phone app inherits
most of it (`practices/native.md` says which parts), and the desktop launcher
follows it for anything React. Where this file and a library's official
documentation disagree on an API, the documentation wins; this file is house
style on top of current APIs.

---

## Table of contents

1. [Technology stack](#technology-stack)
2. [Product rules (read first)](#product-rules-read-first)
3. [Frontend commandments (ENFORCED)](#frontend-commandments-enforced)
4. [Project structure](#project-structure)
5. [Naming conventions](#naming-conventions)
6. [Imports](#imports)
7. [Components](#components)
8. [Data fetching (TanStack Query)](#data-fetching-tanstack-query)
9. [State management (Zustand)](#state-management-zustand)
10. [Forms (React Hook Form + Zod)](#forms-react-hook-form--zod)
11. [API layer (REST over the HTTP gateway)](#api-layer-rest-over-the-http-gateway)
12. [Live events (WebSocket client)](#live-events-websocket-client)
13. [Topology canvas (React Flow)](#topology-canvas-react-flow)
14. [UI components (shadcn)](#ui-components-shadcn)
15. [Styling and theming (Tailwind)](#styling-and-theming-tailwind)
16. [Error handling](#error-handling)
17. [Routing (react-router)](#routing-react-router)
18. [Utilities](#utilities)
19. [Models and types](#models-and-types)
20. [TypeScript strictness](#typescript-strictness)
21. [Testing (frontend)](#testing-frontend)
22. [Accessibility](#accessibility)
23. [Performance](#performance)
24. [Quick reference: creating a new entity](#quick-reference-creating-a-new-entity)
25. [Quick reference: adding a new UI component](#quick-reference-adding-a-new-ui-component)
26. [Lint & typecheck](#lint--typecheck)

---

## Technology stack

| Category | Technology | Notes |
|---|---|---|
| Framework | React + Vite | Automatic JSX runtime, do **not** `import React` |
| Language | TypeScript (strict, `verbatimModuleSyntax`) | `import type` for type-only imports |
| Router | react-router | `RouterProvider` from `react-router/dom`, everything else from `react-router`; `react-router-dom` is never used |
| State | Zustand | `useShallow` for multi-field selectors |
| Data fetching | TanStack Query | Object-form `useQuery`/`useMutation`, `isPending` over `isLoading` |
| Forms | React Hook Form + Zod | Schemas co-located with models |
| UI primitives | shadcn/ui on the unified `radix-ui` package | React-19-native (no `forwardRef`) |
| Headless primitives | `@shadcn/react` | Separate install path from the shadcn CLI, used for `message-scroller` |
| Styling | Tailwind CSS | `@theme inline` + CSS variables |
| Topology canvas | `@xyflow/react` (React Flow) | Canvas JSON holds nodes, edges, and positions |
| Canvas layout | `elkjs` | Dependency-aware auto-layout on first paint |
| Rich text | tiptap, with `yjs` and `y-protocols` | Collaborative document editor, ADR 0026 and ADR 0050 |
| Voice | LiveKit | ADR 0030 |
| Drag and drop | dnd-kit | Board drag and drop, ADR 0001 |
| HTTP (REST) | Axios | Typed generics + auth interceptor |
| Live events | native `WebSocket` client | Separate from Axios, see [Live events](#live-events-websocket-client) |
| Notifications | Sonner | |
| Icons | lucide-react | |
| Async dialogs | react-confirm | Promise-returning dialogs via `useConfirmationDialog` / `useFormDialog` |

Versions are pinned in `web/package.json`; this table names the choice, not the version.

---

## Product rules (read first)

These are the rules that are unique to this product and that override generic
React advice. They exist so the frontend stays a thin, correct peer of the
MCP-first backend.

1. **The browser talks to the HTTP/JSON gateway, never to MCP.** The backend
   exposes one domain/use-case layer through two adapters: an MCP server (for
   LLMs) and an HTTP/JSON gateway (for this app). The browser uses the gateway
   via Axios ([API layer](#api-layer-rest-over-the-http-gateway)). Do **not**
   speak JSON-RPC from the browser. This is ADR 0019.
2. **Live updates come over one WebSocket, not polling.** Runner status,
   deploy progress, topology mutations and entity changes arrive on a single
   WS connection ([Live events](#live-events-websocket-client)) and are
   fanned into Zustand and TanStack Query. Do not poll for status that the
   server already pushes.
3. **Third-party credentials go through the ticker, never a bare Save.** Any
   form that hands a token, secret, or app registration to an outside service
   (Cloudflare token, GitHub App, LiveKit keys) is a ticker: a list of named
   checks with a why under each, a Verify button that runs one request per
   check in parallel, and a Continue/Confirm that only unlocks once every row
   is green. Build it from `useTicker` (`hooks/useTicker.ts`) + `TickerRow`
   (`components/TickerRow.tsx`); the backend exposes the checks as
   `POST …/verify?check=<key>` returning 204, 200 with a `detail` line shown
   under the green row (the domains a Cloudflare token can edit), or the
   provider's error, and stores nothing. Examples: `ManualConnectorDialog`,
   `GitHubAppForm`.
   Never collapse the checks into one request: a single red line can't tell
   the user which permission is missing.

---

## Frontend commandments (ENFORCED)

> These are **hard rules**, not suggestions. A PR that breaks them is not
> done, regardless of what the rest of the codebase currently looks like.
> **Do not copy a nearby file's pattern if it violates this section.** Copy
> the rule, not the drift; copying code that breaks a rule is not a defense.

### F1: component hierarchy, Page to Feed to Section to Card

Every screen decomposes into named, single-responsibility components:

- **`XxxPage`**: fetches, handles loading/error/empty, composes Feeds. No
  data transforms, no inline lists.
- **`XxxFeed`**: the list of entities (search/filter/actions). Lists are
  hairline rows, never a table library, see the design language's
  [Pattern spec](design-language.md#pattern-spec) (List and row).
- **`XxxSection`**: a titled group rendered inside a page or feed.
- **`XxxCard`/`XxxRow`/`XxxItem`**: one repeated entity.

**DO NOT** write an inline `.map()` whose body is a `<section>` or any JSX
block larger than a trivial `<li>`/`<option>`. Extract it into a named
component and render that.

```tsx
// banned: inline .map rendering sections/cards
{swimlanes.map((lane) => (
  <section key={lane.key}> ... 40 lines of nested JSX ... </section>
))}

// required: named components
{swimlanes.map((lane) => <SwimlaneSection key={lane.key} lane={lane} />)}
```

### F2: conditional rendering with `&&` (negative checks first)

Render conditional states with `&&` blocks, ordered negative-first
(loading, then error, then empty, then data). **DO NOT** use
`if (x) return <Component/>` early-returns for rendering, and **DO NOT** use
ternaries to pick between components (see F4). Keep the conditions mutually
exclusive: with TanStack Query they already are (`isPending`, `error`, and
`data` never overlap). Pages keep this explicit pattern rather than React
19's `use` plus `Suspense`, for clarity and one uniform error UI.

```tsx
// required
return (
  <Container>
    {isPending && <LoadingDisplay />}
    {error && <ErrorDisplay error={error} />}
    {data && data.length === 0 && <NoDataDisplay message="No tickets yet" />}
    {data && data.length > 0 && <TicketsFeed tickets={data} />}
  </Container>
);

// banned: if/return for rendering
if (isPending) return <LoadingDisplay />;
if (error) return <ErrorDisplay error={error} />;
return <TicketsFeed tickets={data ?? []} />;
```

> Watch out: use `data.length > 0 &&`, **never** `data.length &&`. The
> latter renders a literal `0` when the list is empty.
>
> Exception: a bare `if (x) return null` *guard* is fine (it renders
> nothing). The ban is on `if (x) return <SomeComponent/>`. Note that
> `react-confirm` dialogs need **no** `if (!open) return null` guard.
> `confirmable` passes `show` and the dialog renders `<Dialog open={show}>`
> (see the Async dialog section).

#### React Query: don't over-guard (no overkill)

TanStack Query's flags are **mutually exclusive**: `data` is only defined
once the query succeeds, and `isPending`/`error`/success never overlap (per
the TanStack docs: after you've handled pending and error, "we can assume
`isSuccess === true`"). So gate the success branch on `data &&` **alone**,
unless the query passes `placeholderData`. A query with `placeholderData` can
flip `status` to `success` (so `data` is defined) while `isPlaceholderData`
is true and no real fetch has completed; gating on `data &&` alone then
renders placeholder content as if it were the real result. None of the
queries in this codebase use `placeholderData` today, so the rule holds as
written; add the `!isPlaceholderData` check the day one does.
Do **not** stack `!isPending && !error &&` in front of it: that is redundant
overkill.

```tsx
// required: single object, data is undefined until success, so this is complete
const { data: doc, error, isPending } = useFetchDoc(docId);
{isPending && <LoadingDisplay />}
{error && <ErrorDisplay error={error} />}
{doc && <DocDetail doc={doc} />}

// required: list, do NOT default to []; gate on data &&
const { data: services, error, isPending } = useFetchServices(projectId);
{isPending && <LoadingDisplay />}
{error && <ErrorDisplay error={error} />}
{services && services.length === 0 && <NoDataDisplay message="No services yet" />}
{services && services.length > 0 && <ServicesFeed services={services} />}

// banned: overkill, redundant guards
{!isPending && !error && doc && <DocDetail doc={doc} />}
```

**The `= []` trap.** If you destructure with a default (`data: services = []`),
`services.length === 0` is true *while loading too*, so the empty state
flashes unless you also gate on `!isPending`. Avoid the trap: **don't default
query data to `[]`** when you render a length-based empty state; gate on
`services &&` instead. Keeping `= []` is only fine when you merely map over the
data in a computation and render no length-based empty state.

**No bare fragments around the blocks.** Don't wrap the loading/error/empty/
data blocks in `<>…</>`. They already sit inside the page's `Container`; if a
group genuinely needs its own wrapper, extract a `XxxSection` component instead
of a fragment.

#### React Query: `isPending` is the house flag (never `isLoading`)

Every example in this guide destructures `isPending`; that is the house
standard for query loading state. In TanStack Query v5 `isLoading` is derived
(`isPending && isFetching`), so it is **false** for a disabled query
(`enabled: !!id`, which this guide mandates for detail queries) that has never
fetched, and the loading branch silently never renders. Do not use `isLoading`
for queries.

### F3: use the shared display components

`LoadingDisplay`, `ErrorDisplay`, `NoDataDisplay`, `Container`, all in
`components/`. **DO NOT**
hand-roll `<p>Loading…</p>`, `<p>Failed to load.</p>`, or inline empty-state
`<p>` blocks. If a state has no shared component, build one in
`components/`; don't inline it.

### F4: `&&` for components, ternaries only for values

Use `&&` to show/hide components. **Never use a ternary to choose between
components**: split it into two `&&` blocks. Ternaries are allowed only when
they produce a *value* (a string, class, or number), never JSX. Never nest
ternaries.

```tsx
// banned: ternary choosing between components
{repos.length === 0 ? <NoDataDisplay message="No repos yet" /> : <RepoList repos={repos} />}

// required: two && blocks
{repos.length === 0 && <NoDataDisplay message="No repos yet" />}
{repos.length > 0 && <RepoList repos={repos} />}

// required: ternary fine here, it produces a value, not a component
const label = isPending ? "Saving…" : "Save";
<span className={cn("text-sm", active && "font-semibold")} />
```

### F5: `useState`/`useEffect` are a last resort

- **DO NOT** fetch data in `useEffect`; use TanStack Query hooks.
- **DO NOT** compute derived values in `useEffect`/`useState`; derive with
  `useMemo` during render.
- **DO NOT** sync fetched data into local state; render from the query cache.
- `useEffect` is for **side effects only** (subscriptions, one-shot DOM/redirect,
  logging). `useState` is for genuine local UI state (open/closed, drag target).
- Server state lives in TanStack Query; cross-surface client state in Zustand;
  form state in React Hook Form; ephemeral UI state stays local to the
  component. Never duplicate server state into `useState`/Zustand.
- Loading and connection state derive from real state (`isPending`, the
  socket's status), never from whether an object or cache entry happens to
  exist.

### F6: no prop drilling

Passing the same data/callbacks through more than 2 levels is banned.
Instead: fetch reference data inside the component that needs it (via a
hook), or read it from a Zustand store. If a child needs `projects`,
`categories`, `ticketTypes`, the child fetches them; the parent does not
hand them down.

### F7: files own one concern, pages stay thin

A page file contains only the page component. Sub-components live in
`components/<domain>/`. Any file over 200 lines (component/page) or 300 lines
(hooks) must be split. An unexported helper component defined inside a page
file is a defect; promote it to `components/<domain>/`.

### Before you mark web work done (self-review)

Run this checklist; every item is a gate:

- [ ] No inline `.map()` rendering a `<section>`/large JSX block (F1)
- [ ] Loading/error/empty rendered with `&&` + shared displays, negative-first (F2, F3)
- [ ] No `if (x) return <Component/>`; no ternaries choosing components; `&&` only (F2, F4)
- [ ] No fetch/derived/sync work in `useEffect`/`useState` (F5)
- [ ] No prop passed more than 2 levels (F6)
- [ ] No file over the size limits; no helper components inside page files (F7)
- [ ] Every API call goes through a typed hook in `hooks/XxxHooks.tsx`, never `api` in a page or component
- [ ] No hard-coded palette classes (`bg-yellow-100`, `text-blue-800`) on themed surfaces; semantic tokens only
- [ ] `bun run --cwd web typecheck`, `lint`, `build`, and tests all green

---

## Project structure

```
web/src/
  components/           # Reusable UI components
    ui/                 # shadcn/ui primitives (CLI-generated)
    topology/           # React Flow nodes/edges (ServiceNode, edges)
    ticket/             # Domain group: TicketFeed, CreateTicketDialog, ...
    doc/
    deploy/
    ...
  hooks/                # Custom hooks + grouped data hooks (XxxHooks.tsx)
  lib/                  # cn() and tiny shared helpers (utils.ts)
  models/               # Interfaces + Zod schemas (.tsx)
  enums/                # String-union / as-const "enums" (.tsx)
  pages/                # Route-level page components
  stores/               # Zustand stores (incl. the React Flow store)
  api/                  # Axios instance, error types, the container log stream
  utils/                # Pure functions (TimeUtility.tsx and friends)
  Layout.tsx            # Root layout: header, <Outlet/>, footer
  main.tsx              # App entry
  Router.tsx            # createBrowserRouter definitions
```

Rules:

- One folder per domain entity under `components/`.
- No barrel/`index.ts` files; import by full path. Barrels defeat
  tree-shaking and hide which way dependencies run.
- `.tsx` everywhere is deliberate house style, even for pure-TS model, enum
  and utility files. Use `import type` for type-only imports so these files
  stay clean under `verbatimModuleSyntax`.
- A module the phone app runs too lives in `client-core/` at the repository
  root and is imported as `@nexul/client-core/<module>` (ADR 0139): the
  permission table, chat and embed rules, people, the live socket, the query
  retry rule. It holds no React and no browser API. Its imports form their own
  group, after the third-party ones.

---

## Naming conventions

| Item | Convention | Example |
|---|---|---|
| Component file | PascalCase | `TicketsFeed.tsx`, `CreateTicketDialog.tsx` |
| Component export | named | `export const TicketsFeed` |
| Props interface | `XxxProps` | `TicketsFeedProps` |
| Hook file | PascalCase, hooks `useXxx` | `TicketHooks.tsx`, `useAuth.tsx` |
| Store file / hook / type | `XxxStore.tsx` / `useXxxStore` / `XxxStore` | `flowStore.tsx` |
| Page file | `XxxPage.tsx` | `TicketsPage.tsx` |
| Model file | PascalCase | `Ticket.tsx` |
| Enum file | PascalCase | `Deploy.tsx` |
| Query-key constant | `getXxxKey` | `getTicketsKey = "getTickets"` |
| Form schema / data | `XxxFormSchema` / `z.infer<...>` | `SaveTicketFormSchema` |
| shadcn primitive | file kebab, default-or-named per CLI | `button.tsx` |
| Utility function | camelCase | `cn()`, `toCamelCase()` |

### Component archetypes

| Pattern | Name | Purpose |
|---|---|---|
| Page header | `PageHeader` | Display-face title (`pageTitleClassFor`, held to three lines by `ClampedTitle`) + one-line subtitle (page-level marquee) |
| List view | `XxxFeed` | Hairline-row list with search, filter, actions |
| Create dialog | `CreateXxxDialog` | New-entity dialog |
| Delete dialog | `DeleteXxxDialog` | Confirmation dialog |
| Edit dialog | `EditXxxDialog` / `AddXxxDialog` | Edit / add related data |
| Form fragment | `XxxFormFragment` | Reusable section via `useFormContext` |
| Form adaptor | `XxxFormAdaptor` | Conditional fields on other values |
| Info display | `XxxInfoDisplay` | Read-only display |
| Async dialog | `useConfirmationDialog` / `useFormDialog` | `react-confirm` promise result |
| Titled group | `XxxSection` | A named group inside a page/feed (F1) |
| Repeated entity | `XxxCard` / `XxxRow` / `XxxItem` | One row/card of a list (F1) |
| Empty state | `NoDataDisplay` | Shared "nothing here" display (F3) |
| Ticker | `useTicker` + `TickerRow` | Named third-party checks that verify before Continue ([Product rules](#product-rules-read-first)) |
| Topology node | `XxxNode` | React Flow custom node ([Topology canvas](#topology-canvas-react-flow)) |

---

## Imports

Order, top to bottom, blank line between groups:

1. Third-party libraries (`@tanstack/react-query`, `zod`, `lucide-react`, …)
2. The client core shared with the phone (`@nexul/client-core/...`)
3. shadcn/ui primitives (`@/components/ui/...`)
4. Local components (`@/components/...`)
5. Local hooks (`@/hooks/...`)
6. Local stores (`@/stores/...`)
7. Local models / enums (`@/models/...`, `@/enums/...`)
8. Local utilities (`@/lib/utils`, `@/utils/...`)

All local imports use the `@/` alias (`./src`). Use `import type` for
type-only imports (required by `verbatimModuleSyntax`):

```tsx
import { useQuery } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { useFetchTickets } from "@/hooks/TicketHooks";
import type { Ticket } from "@/models/Ticket";
import { cn } from "@/lib/utils";
```

Do **not** write `import React from "react"`. With the automatic JSX runtime
(Vite default, React 19) it is unnecessary. Only import named hooks you use:

```tsx
import { useState } from "react";
```

---

## Components

### Basic component

Explicit `interface` props + destructuring. No `type` for props.

```tsx
interface TicketSummaryProps {
  ticket: Ticket;
  compact?: boolean;
}

export const TicketSummary = ({ ticket, compact = false }: TicketSummaryProps) => {
  return (
    <div className={cn("rounded border p-4", compact && "p-2")}>
      <h2 className="font-semibold">{ticket.title}</h2>
    </div>
  );
};
```

### Feed (list view)

A list is a hand-built hairline-row list, never a table library: a column-header
row over a `<ul>` of `border-b`/`divide-y` rows, exactly like the real
`ServicesFeed.tsx`. The design language's
[Pattern spec](design-language.md#pattern-spec) (List and row) is the row
layout of record; this section covers the component shape only.

```tsx
import type { Ticket } from "@/models/Ticket";

interface TicketsFeedProps {
  tickets: Ticket[];
}

export const TicketsFeed = ({ tickets }: TicketsFeedProps) => (
  <div className="mt-2">
    <div className="flex items-center gap-3 border-b border-border px-4 pb-1.5 text-xs text-muted-foreground">
      <span className="flex-1">Title</span>
      <span className="w-20 shrink-0 text-right">Status</span>
    </div>
    <ul className="divide-y divide-border">
      {tickets.map((ticket) => (
        <TicketRow key={ticket.id} ticket={ticket} />
      ))}
    </ul>
  </div>
);
```

`TicketRow` (an `XxxRow`, see F1) renders one `<li>` as a `Link`: primary field
left-aligned, secondary/meta right-aligned in `font-mono text-xs
text-muted-foreground`. Row hover is a background lift only
(`hover:bg-accent/40`); shadow and scale are reserved for draggable cards
(kanban `TicketCard`, `ProjectCard`), not feed rows.

### Async dialog (common hooks over `react-confirm`)

Dialogs are promise-based: the caller `await`s a result instead of juggling
`open` state. This is the **required** dialog pattern. Two common hooks wrap
`react-confirm` so the codebase **never uses the library primitives
directly**.

**Confirmation, `useConfirmationDialog`**: `open({ message, title?,
confirmLabel?, cancelLabel?, destructive? }) → Promise<boolean>` (`true`
confirmed, `false` cancelled, never hangs). The **caller** runs the
mutation on `true`; `destructive` defaults to `true` and switches the
confirm button to the `destructive` variant.

```tsx
// call site
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";

const { open: confirmDelete } = useConfirmationDialog();
const onDelete = async () => {
  const ok = await confirmDelete({ message: "Delete this service?" });
  if (ok) deleteService.mutate(service.id);
};
```

**Form, `useFormDialog`**: `open<TFormValues>({ title, description?, form,
schema, okLabel?, cancelLabel?, formOptions? }) →
Promise<FormDialogResult<TFormValues>>` with `FormDialogResult<T> =
{ success: boolean; data: T | null }` (`success: false` = cancelled). The
**shell owns the React Hook Form instance** (`zodResolver` from the passed
`schema`, `formOptions` forwarded) and renders the form inside a
`FormProvider`; the form consumes `useFormDialogContext<TFormValues>()`,
which merges the RHF methods (`register`, `watch`, `formState`, …) with two
dialog utilities:

- `onSubmit(handler)`: register the mutation. The handler's **return**
  closes the dialog with that data; a **throw** keeps it open and the shell
  auto-surfaces the error (`root.serverError`, unless the form already set
  one).
- `setLoading(bool)`: disables the shell's OK button (e.g. while the form
  loads its initial data).

```tsx
// components/ticket/CreateTicketForm.tsx (used inside a FormDialog)
import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";

export const CreateTicketForm = () => {
  const { register, formState, onSubmit } =
    useFormDialogContext<SaveTicketFormData>();
  const createTicket = useCreateTicket();

  onSubmit(async (input) => {
    const id = await createTicket.mutateAsync(input);
    return { id, ...input }; // return → dialog closes with this data
  });

  return (
    <>
      <input placeholder="Title" {...register("title")} />
      {formState.errors.title && (
        <p role="alert">{formState.errors.title.message}</p>
      )}
    </>
  );
};
```

```tsx
// call site
import { useFormDialog } from "@/hooks/useFormDialog";

const { open: openCreate } = useFormDialog();
const onCreate = async () => {
  const result = await openCreate<SaveTicketFormData>({
    title: "New ticket",
    schema: SaveTicketFormSchema,
    form: <CreateTicketForm />,
  });
  if (result.success && result.data) navigate(`/tickets/${result.data.id}`);
};
```

Rules:

- **Hard rule: never use `react-confirm` primitives directly**
  (`confirmable`, `createConfirmation`, `ContextAwareConfirmation`), always
  via `useConfirmationDialog` / `useFormDialog`. The only `confirmable`
  usages live inside `components/dialogs/`.
- The dialog components (`components/dialogs/ConfirmationDialog.tsx`,
  `FormDialog.tsx`) receive `show` from `confirmable` and render
  `<Dialog open={show}>`; no `if (!open) return null` guard needed (F2).
- `dismiss()` leaves the promise pending forever; every close path goes
  through `proceed` (cancel calls `proceed(false)` / `proceed({ success: false,
  data: null })`).
- Validation lives in the shell's schema (RHF + Zod via `zodResolver`);
  field errors render in the form. The form never renders its own submit
  button; the shell's OK button drives `handleSubmit`.
- The caller handles navigation / invalidation (invalidation usually already
  happens in the hook's `onSuccess`).

### Form fragment

Complex forms split into fragments that read `useFormContext`:

```tsx
import { Controller, useFormContext } from "react-hook-form";
import { Select } from "@/components/ui/select";

interface DeployFormFields {
  strategy: string;
}

export const DeployStrategyFragment = () => {
  const { control } = useFormContext<DeployFormFields>();
  return (
    <Controller
      name="strategy"
      control={control}
      render={({ field }) => (
        <Select value={field.value} onValueChange={field.onChange} />
      )}
    />
  );
};
```

---

## Data fetching (TanStack Query)

All hooks for an entity live in one file (`hooks/TicketHooks.tsx`), and pages
and components call those hooks, never `api` directly.

The app's query client is `lib/queryClient.ts`: a 30-second stale time, since
the socket pushes every change, and no retry for a 4xx, the rule the phone
shares (`@nexul/client-core/queryRetry`), so a missing or forbidden page shows
its not-found screen at once.

| Operation | Hook | Method |
|---|---|---|
| List | `useFetchXxx` | GET list |
| One | `useFetchXxx(id)` | GET one, `enabled: !!id` |
| Create | `useCreateXxx` | POST |
| Update | `useUpdateXxx(id)` | PUT/PATCH |
| Delete | `useDeleteXxx` | DELETE |

### Query hook

```tsx
import { useQuery } from "@tanstack/react-query";
import { api } from "@/api/client";
import type { Ticket } from "@/models/Ticket";

export const getTicketsKey = "getTickets";

export const useFetchTickets = () =>
  useQuery({
    queryKey: [getTicketsKey],
    queryFn: () => api.get<Ticket[]>(`/api/tickets`).then((r) => r.data),
  });
```

### Single-resource query

```tsx
export const useFetchTicket = (ticketId: string) =>
  useQuery({
    queryKey: ["getTicket", ticketId],
    queryFn: () => api.get<Ticket>(`/api/tickets/${ticketId}`).then((r) => r.data),
    enabled: !!ticketId,
  });
```

### Mutation hook

```tsx
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router";
import { toast } from "sonner";
import { api } from "@/api/client";
import { getTicketsKey } from "./TicketHooks";

export const useCreateTicket = () => {
  const client = useQueryClient();
  const navigate = useNavigate();
  return useMutation({
    mutationFn: (input: SaveTicketFormData) => api.post<string>("/api/tickets", input).then((r) => r.data),
    onSuccess: async (id) => {
      await client.invalidateQueries({ queryKey: [getTicketsKey] });
      toast.success("Ticket created");
      navigate(`/tickets/${id}`);
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};
```

Rules:

- Query keys are exported constants (`getXxxKey`); a parameterized key
  includes the param: `["getTicket", id]`.
- Detail queries set `enabled: !!id` so they never fetch with an undefined id.
- Every mutation invalidates the relevant list key in `onSuccess`, and may
  refetch the single resource, unless the response is the full entity and
  the list is high-churn (chat messages): then `setQueriesData` patches it in
  place, with an optimistic row from `onMutate` where the user expects
  instant feedback.
- `toast.success` on success, `toast.error(errorMessage(error))` on error
  ([Error handling](#error-handling)). A form saved from a settings card's
  strip (`SettingsSaveBar`) answers in its Save button instead, so its hook
  sends no success toast.
- The server pushes changes over the socket ([Live events](#live-events-websocket-client)); never poll.
- Do **not** default list data to `= []` when you render a length-based
  empty state (see the "React Query: don't over-guard" callout in F2).
- A query used in more than one place (a component plus a prefetch, or two
  hooks reading the same resource) is defined once with `queryOptions()` and
  shared between them. One key and one fetcher in one place means the two
  call sites cannot drift out of sync with each other.
- `hooks/TicketCache.tsx` is the only code that reads or writes a ticket view
  in the cache. It owns the four views (all, one, by doc, by project) and
  its operations, `ticketChanged`, `ticketCreated`, `ticketRemoved`,
  `ticketCategoryMoved` and `statusRemoved`; every mutation, board drop,
  category move, test report and live ticket frame goes through them. A hook
  that patches or invalidates one ticket key by itself leaves the other three
  views stale.
- A batch read for the items on a page (the board's run states and thread
  markers) is keyed by the project, never by every item id. Ids in the URL
  cross nginx's 8KB header buffer at around 220 tickets, and the key changes
  with every new ticket. The endpoint takes `project_id` and still checks
  each target it returns.

---

## State management (Zustand)

### The three tiers of state

| Tier | Tool | What lives here |
|---|---|---|
| Server state | TanStack Query | Anything fetched from the API |
| Client UI state | Zustand | Theme, sidebar open/closed, canvas viewport |
| Form state | React Hook Form | In-progress form values |

Server state and form state never go in Zustand (F5).

### Basic store

```tsx
import { create } from "zustand";

export type ThemeStore = {
  theme: "dark" | "light";
  setTheme: (theme: "dark" | "light") => void;
};

export const useThemeStore = create<ThemeStore>((set) => ({
  theme: "dark",
  setTheme: (theme) => set({ theme }),
}));
```

### Persisted store, partial state

Persist data, not transient UI flags, via `partialize`:

```tsx
import { create } from "zustand";
import { persist } from "zustand/middleware";

export type SessionStore = {
  token?: string;
  isLoggedIn: boolean;
  lastViewed?: string;
  login: (token: string) => void;
  logout: () => void;
};

export const useSessionStore = create<SessionStore>()(
  persist(
    (set) => ({
      isLoggedIn: false,
      login: (token) => set({ isLoggedIn: true, token }),
      logout: () => set({ isLoggedIn: false, token: undefined }),
    }),
    { name: "session", partialize: (s) => ({ token: s.token }) },
  ),
);
```

### Multi-field selectors: use `useShallow`

Selecting an object with several fields **must** use `useShallow`, or the
component re-renders on every store update (new object identity each time):

```tsx
import { useShallow } from "zustand/react/shallow";

const { theme, setTheme } = useThemeStore(
  useShallow((s) => ({ theme: s.theme, setTheme: s.setTheme })),
);
```

Store rules:

- Export the hook (`useXxxStore`) and the state type (`XxxStore`).
- Use `persist` + `partialize` for localStorage-backed *data*.
- Outside React, read with `useXxxStore.getState()` (no re-render).
- Clear on logout: `useSessionStore.persist.clearStorage()`.

---

## Forms (React Hook Form + Zod)

Form schemas live in the model file ([Models and types](#models-and-types)).
Forms go through React Hook Form and Zod rather than React 19's form Actions
and `useActionState`, so validation stays typed end to end and field errors
map onto the shared error envelope
([API layer](#api-layer-rest-over-the-http-gateway)) instead of a plain
action-state string. Zod is the schema library; which major version of it the
repo runs is a decision for the whole codebase, never a per-file choice.

Setup + submit:

```tsx
const form = useForm<SaveTicketFormData>({
  defaultValues: { title: "", body: "" },
  resolver: zodResolver(SaveTicketFormSchema),
});

<form onSubmit={form.handleSubmit((data) => mutateAsync(data))} className="space-y-4">
  <FormInput control={form.control} name="title" label="Title" />
  <FormInput control={form.control} name="body" label="Body" />
  <Button disabled={form.formState.isSubmitting} type="submit">
    {form.formState.isSubmitting ? "Saving..." : "Save"}
  </Button>
  {form.formState.errors.root && (
    <p className="text-destructive">{form.formState.errors.root.message}</p>
  )}
</form>;
```

Form rules:

- Use `FormInput` / `FormSwitch` for standard fields; raw `FormField` for the
  rest.
- Compose big forms with `useFormContext` fragments.
- Disable submit with `form.formState.isSubmitting`.
- Surface server-side root errors via `setError("root", ...)` in `onError`.

---

## API layer (REST over the HTTP gateway)

The browser uses **one** Axios instance, `api` in `api/client.tsx`, pointed at
the Go HTTP/JSON gateway ([Product rules](#product-rules-read-first)). Axios is
the choice here, not a thin `fetch` wrapper, because the client leans on its
request and response interceptors for the two things this app needs: bearer
token injection on every request and one place to catch a global 401. A
thinner client would need the same hooks wired up by hand for no gain at this
app's scale.

`ApiErrorBody` **matches the backend error envelope**
(`internal/platform/httpx`): a `message`, a stable machine `code`, optional
per-field `errors`, and optional `details`. The parser in
[Error handling](#error-handling) reads exactly this.

`api` does not unwrap `.data`: type the response with a generic and read
`.data` off the promise.

```tsx
const tickets = await api.get<Ticket[]>("/api/tickets").then((r) => r.data);
const id = await api.post<string>("/api/tickets", input).then((r) => r.data);
```

> Dev mode: the gateway serves the SPA same-origin in production (embedded
> webui) and sends no CORS headers, so `vite.config.ts` proxies `/api`,
> `/auth`, and `/ws` to `http://localhost:8080`. Run the dev server with
> `VITE_API_URL=/`.

API rules:

- Always use `api`, never raw `axios` (except one-off OAuth redirects).
- Always type responses with generics.
- The request interceptor injects the bearer token, read with `getState()`
  (the setup pass stands in before the first user exists).
- Handle errors at the hook or component layer, not in the interceptor. The
  interceptor only handles a 401: logout, then redirect to `/login`.

---

## Live events (WebSocket client)

Status the server pushes (runner heartbeats, deploy progress, topology
mutations, ticket, doc and chat changes, dead-letter alerts) arrives on
**one** WebSocket, separate from Axios. `LiveEventsClient` in
`@nexul/client-core/liveSocket` (shared with the phone, ADR 0139) owns the connection;
`hooks/useLiveEvents.tsx` mounts it once at the layout root, routes each typed
frame (`{ topic, type, payload }`) to every domain that follows its topic, and
keeps the one rule that crosses domains: when the viewer's own permissions
move, every open read refetches (ADR 0134).

Each domain follows its own events. Its hooks file exports one live follower
beside its keys and queries (`ticketFollower` in `TicketHooks.tsx`,
`docFollower` in `DocHooks.tsx`; chat's is `ChatFollower.tsx`, beside
`ChatHooks.tsx`), and the follower is added to the list in `useLiveEvents`. A
`LiveFollower` (`lib/live.ts`) maps each topic it follows to a function of the
frame's payload and a `Live` holding the query client and the router:

```tsx
export const categoryFollower: LiveFollower = {
  "category.created": ({ category }: CategoryPayload, { client }) =>
    client.invalidateQueries({ queryKey: [getProjectCategoriesKey, category.project_id], exact: true }),
  "category.updated": ({ category }: CategoryPayload, { client }) =>
    replaceRow(client, [getProjectCategoriesKey, category.project_id], category),
  "category.deleted": ({ category }: CategoryPayload, { client }) =>
    dropRow(client, [getProjectCategoriesKey, category.project_id], category.id),
};
```

Rules:

- Mount once (layout root). Components subscribe to stores and queries, never
  to the socket.
- Reconnect with exponential backoff and jitter, and after a reconnect refetch
  every open read, since the frames sent while the socket was down are lost.
  The socket stays open while the tab is hidden; the phone closes its own in
  the background.
- A follower touches only its own domain's cache. When another domain knows
  what a frame changed for it, the follower calls that domain's cache
  operation (`ticketChanged`, `stageMoved`), never its keys. Several domains
  may follow one topic, each for its own views.
- Scope by the ids in the payload. A frame that carries the entity patches it
  into the views holding it and refetches nothing: a ticket body commit
  arrives every 5 seconds while someone types, and patched it costs no
  request. A frame that only names the entity invalidates that entity's
  queries and the cached lists that hold it (`refetchHolding`), never a whole
  key prefix. A row that would move in its list, or that no list holds yet,
  refetches the list for the server's order (`replaceRow`).
- When a frame lacks an id a view is keyed by, the follower falls back to
  every view of that kind and says so in a comment; the fix is an additive
  field on the server's event.
- Payload interfaces live with the follower that reads them, never in a
  shared live module.
- Test a follower through its interface: hand it a frame with `followFrame`
  (`test/followFrame.tsx`) and assert which cached data changed and which went
  stale. A change that cuts requests also gets a count in
  `hooks/liveRequests.test.tsx`, which mounts real views beside the socket
  and lists the GETs one frame sends.
- The WS client **never** mutates server state; it is read-side only. Writes
  go through the REST gateway, which stays the source of truth on read.
- Carry a `trace_id` on frames where present; log it via the same structured
  logger convention as the backend.

### The live topic contract

The server bridges the bus topics in `livePushTopics` (`server/cmd/main.go`)
onto the socket, and its audience rules (`liveRules` in
`server/cmd/live_audience.go`) decide who receives each; a topic without a
rule reaches nobody, and `TestLiveRules_MatchWhatIsPushed` fails when the two
lists disagree. `make live-topics` writes the rules' topics to
`web/src/hooks/liveTopics.generated.json`, and two more tests hold the
server and the browser together:

- `TestLiveTopicsFile_MatchesTheRules` (Go) fails when the generated file is
  stale, the way `sqlc diff` does.
- `hooks/liveTopics.test.tsx` fails when the followers follow a topic the
  server never pushes, or when a pushed topic is followed by no follower and
  not listed in its `ignoredTopics` with the reason.

Adding a live topic is four steps in one change:

1. Publish it from the domain (`events.go`, `practices/architecture.md`
   section 2).
2. Bridge it in `livePushTopics` and add its audience rule to `liveRules`,
   checked through the permission table the entity's own read uses, so a
   socket never receives what its person could not load.
3. Run `make live-topics` and commit the regenerated JSON.
4. Follow it in the domain's follower (patch from the payload, or invalidate
   by its ids), and in the phone query's `refreshes` (`practices/native.md`
   section 4) if the phone shows the entity.

---

## Topology canvas (React Flow)

The deploy topology is rendered with `@xyflow/react`. Visual tokens (canvas
field, node cards, edges, selection ring) live in the shared theme; see
[Canvas (topology)](design-language.md#canvas-topology) (the spec of record)
and `web/src/index.css` (the token source of record); this section covers
mechanics only. The nodes are ours and carry Nexul-specific signals (MCP
index health, owning ticket, live deploy status).

- **The canvas store.** Node and edge state lives in an external Zustand
  store, `stores/flowStore.tsx`, which holds `nodes`, `edges`, the React Flow
  change handlers (`applyNodeChanges`, `applyEdgeChanges`, `addEdge`) and
  `applyServerPatch` for the patches [Live events](#live-events-websocket-client)
  push. React Flow performs best this way; components select slices
  (`useShallow` for several fields) so only changed nodes re-render. The
  canvas mounts inside a `ReactFlowProvider` (`TopologyCanvas.tsx`,
  `TopologyFlow.tsx`).
- **One flexible node per kind.** A single `ServiceNode` covers every service
  card; its icon and footer slots are driven by `data`, not by a component
  per service type.
- **Typed edges carry meaning.** Edges are typed (`depends_on`,
  `connects_to`, `mounts`) because that topology is exactly what the MCP
  server answers with ("what breaks if I redeploy postgres?"). They render
  dashed on a `smoothstep` path (`RelationEdge.tsx`).
- **The canvas JSON is the infra model, for what it stores (ADR 0033).**
  `getNodes()` / `getEdges()` serialized holds nodes, edges, and positions;
  that is the topology the backend stores and the MCP server mutates.
  Validate it against the shared schema on load and save; never keep a
  second hand-written topology type. The dashed network boxes, gateway rows,
  and hostname pills on the canvas are derived at render time from that
  stored data, never serialized. First paint uses `elkjs` for
  dependency-aware auto-layout; manual drags persist as positions.
- **Theming.** React Flow's CSS is overridden with the same variables as
  shadcn, so nodes, edges, and the controls follow light and dark for free;
  a selected node takes the shadcn `ring`. The canvas has no minimap and
  hides the library's attribution (`proOptions`). Node detail (logs, env,
  deploy history, SSH and docker config) opens in a shadcn `Sheet` on click:
  canvas is overview, sheet is depth.

---

## UI components (shadcn)

Primitives live in `components/ui/`, generated by the shadcn CLI. On React 19
they are written **without `forwardRef`**, `ref` is a normal prop:

```tsx
import type { ButtonHTMLAttributes } from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "@/lib/utils";

const buttonVariants = cva("inline-flex items-center justify-center rounded-md", {
  variants: {
    variant: { default: "bg-primary text-primary-foreground", outline: "border" },
    size: { default: "h-9 px-4", sm: "h-8 px-3", icon: "h-9 w-9" },
  },
  defaultVariants: { variant: "default", size: "default" },
});

export interface ButtonProps
  extends ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {}

export const Button = ({ className, variant, size, ref, ...props }: ButtonProps) => (
  <button ref={ref} className={cn(buttonVariants({ variant, size, className }))} {...props} />
);
```

- Add primitives with `bunx shadcn@latest add <name>`.
- Compose classes with `cn()`.
- A button that starts a request takes `loading={mutation.isPending}` (or
  `isSubmitting`): it disables, sets `aria-busy`, and swaps its leading icon for
  a spinner. Keep a busy label only when the wait is long and worth naming.
- Icons from `lucide-react`: `<Plus className="h-4 w-4" />`.
- Named-export custom components; shadcn primitives follow whatever the CLI
  emits (it now generates React-19-compatible code).
- Enter/exit motion on overlay primitives (dialog, sheet, popover, menu,
  select, hover card, tooltip) is the surface's class (`dialog-surface`,
  `sheet-surface`, `float-surface`, `tooltip-surface`, `overlay-scrim`) and
  its transitions in `index.css`; a new overlay takes one of them, never
  `animate-in`/`animate-out` keyframes, which restart on a reopen. Page, list
  and highlight motion goes through the shared primitives (`EnterList`, `ActiveIndicator`,
  `usePageEntrance`, `lib/motion.ts`); the numbers are the design language's
  [Motion baseline](design-language.md#motion-baseline). The global `prefers-reduced-motion` block in
  `index.css` flattens CSS motion; the primitives keep a short fade.

The shadcn CLI now defaults a fresh `init` to Base UI, not Radix; this repo
stays on the unified `radix-ui` package (`components/ui/dialog.tsx` and every
other primitive import from `radix-ui`, not per-primitive `@radix-ui/react-*`
packages). The `add` command itself has no base-selection flag (only `init`
does: `-b`/`--base <base, radix, aria>`), so before committing a newly added
primitive, run the add with `--dry-run` and check its Dependencies list names
`radix-ui`, not a Base UI package. `@shadcn/react` is a separate install path
from the CLI: shadcn's headless-primitives line, already used for
`message-scroller.tsx`.

---

## Styling and theming (Tailwind)

Tailwind v4 is configured in CSS, not a JS config: `@theme inline` exposes
the CSS variables to utilities, plus a class-based dark variant. This section
covers the mechanics only. The tokens themselves (surfaces and the `panel`
utility, the `brand` accent, status hues, elevation, motion curves, the type
stack, the radius scale) are specified in the design language's
[Shared core](design-language.md#shared-core) and valued in
`web/src/index.css`; how pages sit in panels is in
[What the web app adapts](design-language.md#what-the-web-app-adapts). Fonts
are bundled locally through `@fontsource-variable`, never a CDN.

Use container queries (`@container`, `@min-*`/`@max-*`) for component-level
responsiveness, not viewport breakpoints, whenever a component's layout
should react to the space it is actually given (a card in a narrow sidebar
versus the same card in a wide panel). The layout law is that a
component adapts to its container, not to the viewport it happens to be
mounted in today.

```css
@import "tailwindcss";
@import "tw-animate-css";
@import "@xyflow/react/dist/style.css";

@custom-variant dark (&:where(.dark, .dark *));

@theme inline {
  --color-background: var(--background);
  --color-foreground: var(--foreground);
  --color-card: var(--card);
  --color-primary: var(--primary);
  --color-ring: var(--ring);
  --color-success: var(--success);
  --color-warning: var(--warning);
  --color-info: var(--info);
  --font-sans: var(--font-sans-stack);
  --font-mono: var(--font-mono-stack);
  --font-display: var(--font-display-stack);
  --shadow-card: var(--shadow-card);
  --ease-standard: cubic-bezier(0.25, 0.1, 0.25, 1);
  /* ...the rest of the shadcn tokens */
}
```

- Toggle dark mode by adding/removing `.dark` on `<html>` (the
  `@custom-variant` above makes class-based dark mode work in v4; without it,
  v4 defaults to `prefers-color-scheme`). Startup applies **one coalesced
  write**: an inline script in `index.html` reads the persisted store or
  defaults to **dark** before first paint (dark is the default);
  `themeStore` initialises from the resulting class so it never disagrees
  with the DOM.
- All colors are semantic tokens; never hard-code palette classes for themed
  surfaces. Which roles the `brand` accent and the status hues may take is
  in [The accent](design-language.md#the-accent) and
  [Status](design-language.md#status).
- Conditional classes via `cn()`:

```tsx
<div className={cn("rounded p-4", active && "bg-accent", className)} />
```

- Wrap page content in `Container` (`mx-auto w-full max-w-7xl`).
- The React Flow canvas reads these same variables
  ([Topology canvas](#topology-canvas-react-flow)): one theme, two surfaces.

---

## Error handling

Three levels, plus one shared parser, `errorMessage(error)` in
`api/client.tsx`, which returns the first field error, else the envelope's
`message`, else the transport message:

1. **API**: the gateway returns `{ message, code, errors? }`; the parser
   reads it.
2. **Hook**: every mutation toasts `errorMessage(error)` on error.
3. **Component**: pages render `<ErrorDisplay error={error} />`.

Auth: a 401 in the Axios interceptor clears the session store and redirects;
the WS client closes and reconnects after re-auth.

---

## Routing (react-router)

`RouterProvider` is imported from `react-router/dom`: that entry wires up
react-dom's `flushSync` for you. Everything else (`createBrowserRouter`,
`Outlet`, `useNavigate`, `RouteObject`, and the rest) is imported from
`react-router`. `react-router-dom` is never used. Use a data router with a
layout route that renders `<Outlet/>`.

If vitest resolves `react-router` and `react-router/dom` to two separate
module instances, that duplicates the router context and crashes the app
with "useRouteError must be used within a data router." Fix that with
`resolve.dedupe: ["react-router"]` in both `vite.config.ts` and
`vitest.config.ts`, not by avoiding the `react-router/dom` import.

```tsx
import { createBrowserRouter, type RouteObject } from "react-router";
import { RouterProvider } from "react-router/dom";
import { Layout } from "@/Layout";
import { HomePage } from "@/pages/HomePage";
import { TicketsPage } from "@/pages/TicketsPage";
import { ErrorPage } from "@/pages/ErrorPage";
import { useSessionStore } from "@/stores/sessionStore";

const buildRoutes = (loggedIn: boolean): RouteObject[] => [
  {
    element: <Layout />,
    children: [
      { path: "/", element: <HomePage /> },
      ...(loggedIn ? [{ path: "/tickets", element: <TicketsPage /> }] : []),
      { path: "*", element: <ErrorPage /> },
    ],
  },
];

export const AppRouter = () => {
  const loggedIn = useSessionStore((s) => s.isLoggedIn);
  return <RouterProvider router={createBrowserRouter(buildRoutes(loggedIn))} />;
};
```

- All routes nested under `<Layout/>` (header + `<Outlet/>` + footer).
- The shell is a **sidebar layout**: a sticky left rail
  (`components/sidebar/Sidebar.tsx`). Its header holds the logo, the update
  button, and the collapse toggle (instant width swap, no layout animation,
  see the motion rules). Below it: the workspace switcher; one scrolling nav
  with Search (the command palette) and Inbox, then one project at a time
  behind a project switcher, its pages listed once, then the foldable
  workspace section (Runners, Topology, Automations, Configuration), then the
  conversations (channels, voice channels, direct messages, threads); the
  account menu (Support, Sign out) and the Your settings gear at the bottom.
  The icon rail keeps every page and adds one Chat link for the conversations. The
  signed-out pages and the wizards render without it. Pages render inside `<main>` under
  `Container` (`mx-auto w-full max-w-7xl`).
- Every page reached from a workspace's sidebar lives under `/:workspace`, the
  workspace's slug (ADR 0089); personal and instance pages (`/settings/*`,
  `/login`, `/invite`, `/setup`, the onboarding wizards) stay unprefixed.
  `WorkspaceScope` resolves the slug and the store follows the URL, never the
  other way round. Path builders in `models/` return workspace-relative paths
  (`ticketPath`, `boardPath`); a link wraps them with `useWorkspacePath`
  (`wsPath(boardPath(project))`). A moved or renamed path is not redirected:
  the old one is not found.
- A tabbed page's route ends in `/:tab?` (`automations/:id/:tab?`,
  `settings/:section?/:tab?`), and `PageTabs` reads it; links build tab paths
  with `useTabPath`. Where `:tab` would swallow a sibling's `:id`
  (`automations/hosts` beside `automations/:id`), the route table lists each
  tab as a static route through `staticTabs`, which also wraps the bare path
  so a tab switch keeps the page mounted. `?tab=` is not read.
- Catch-all `*` renders `ErrorPage`, last.
- Auth-gated routes added conditionally.
- Permission-gated areas are listed once in `@nexul/client-core/permissions`,
  which the phone reads too; `models/Access.tsx` adds the web's route areas and
  Settings sections. A sidebar
  entry renders only when the viewer holds its read permission (an Owner's
  `/me` is the whole grid), a route declares its area as `handle` and
  `AreaGate` renders the not-found screen when it can't be opened, and a
  detail page's 403 or 404 lands on the same screen. Create actions follow
  the domain's write permission and are hidden, not disabled, without it.

---

## Utilities

`lib/utils.ts` exports only `cn`:

```ts
import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

export const cn = (...inputs: ClassValue[]) => twMerge(clsx(inputs));
```

Pure helpers go in `utils/` as named files (`TimeUtility.tsx` and friends),
exporting pure functions only, no hooks, no JSX. Dates are formatted by
`TimeUtility.tsx`; there is no separate date library. House style keeps
these `.tsx` ([Project structure](#project-structure)); use `import type` so they stay clean.

---

## Models and types

Model files pair an interface with its Zod schema and the inferred form type;
form types are always `z.infer<typeof Schema>`, never duplicated by hand:

```tsx
import { z } from "zod";

export interface Ticket {
  id: string;
  title: string;
  body: string;
  createdAt: string;
}

export const SaveTicketFormSchema = z.object({
  title: z.string().min(1),
  body: z.string().min(5),
});

export type SaveTicketFormData = z.infer<typeof SaveTicketFormSchema>;
```

### Enums as string unions and `as const`

The Go backend sends enums as strings, so the frontend mirrors them as string
unions, never numeric TS enums (numeric enums drift, aren't tree-shakeable,
and don't match the wire format):

```tsx
export const DeployStatus = {
  Pending: "pending",
  Running: "running",
  Healthy: "healthy",
  Failed: "failed",
} as const;

export type DeployStatus = (typeof DeployStatus)[keyof typeof DeployStatus];
```

Rules:

- `.tsx` for model/enum files (house style).
- Co-locate the Zod schema with the interface it validates.
- Always derive form types with `z.infer`.
- One focused interface per file.

---

## TypeScript strictness

- `strict: true` in tsconfig.
- `verbatimModuleSyntax: true`: forces `import type` for type-only imports.
- `noUncheckedIndexedAccess: true`: array/record access returns
  `T | undefined`.
- `exactOptionalPropertyTypes: true`: `foo?: string` means
  `string | undefined`, not "can be omitted".
- No `any`. Use `unknown` + narrowing. If you must escape, use
  `// eslint-disable` with a reason comment.

---

## Testing (frontend)

### Framework

- **Vitest** (v8 coverage provider) + **React Testing Library** +
  **@testing-library/user-event**. Config in `web/vitest.config.ts`.

### Query priority (RTL)

Query elements the way users find them:

1. `getByRole` (accessible to everyone)
2. `getByLabelText` (form fields)
3. `getByPlaceholderText` (inputs without labels)
4. `getByText` (non-interactive content)
5. `getByDisplayValue` (current form values)
6. `getByAltText` (images)
7. `getByTitle` (last resort)
8. `getByTestId` (only when nothing else works)

Query through `screen`, never through the container or a class name.

### userEvent over fireEvent

`userEvent` simulates real browser behavior (focus, blur, key events);
`fireEvent` dispatches a single DOM event. Always use `userEvent`:

```tsx
import userEvent from "@testing-library/user-event";

const user = userEvent.setup();
await user.click(screen.getByRole("button", { name: "Submit" }));
await user.type(screen.getByLabelText("Title"), "My ticket");
```

### Async assertions

Use `findBy*` (returns a promise) for elements that appear after an async
operation. Avoid `waitFor` + `getBy*` unless you need several assertions in
one wait.

### What to test

- **Hooks:** render with `renderHook`, call actions, assert return values.
- **Components:** render, interact, assert visible behavior. Do not assert
  internal state or CSS classes.
- **Stores:** call actions directly, assert state transitions.
- **Pages:** integration test the full load, display, interact, mutate flow,
  mocking the API layer directly rather than a network-mocking library:
  `vi.mock("@/api/client", () => ({ api: { get: mocks.get } }))` with the
  mock functions built via `vi.hoisted(() => ({ get: vi.fn() }))` so they
  exist before the mock factory runs, then driven per test with
  `vi.mocked(api.get).mockResolvedValue(...)`. Assert the whole flow: load,
  display, interact, mutate, toast. See `TopologyHooks.test.tsx` for the
  full pattern.

### What NOT to test

- shadcn primitives (they have their own tests upstream).
- That `cn()` merges classes correctly (it's `twMerge(clsx(...))`).
- Anything on the cross-language list in `practices/testing.md` section 4.

### Coverage config

The 80% gate is measured by Vitest's v8 provider. Thresholds and the exclude
list live in `web/vitest.config.ts`; read that file directly, since a copy
printed here would drift from the real config.

---

## Accessibility

- Every interactive element must have an accessible name (aria-label or
  visible text).
- Use semantic HTML (`<button>`, `<nav>`, `<main>`, `<dialog>`).
- Focus management: when a dialog opens, focus the first input; when it
  closes, return focus to the trigger.
- Color contrast: meet WCAG AA (4.5:1 for text, 3:1 for large text).
- `prefers-reduced-motion`: disable non-essential animations.

---

## Performance

A performance audit checks three things first: too much data over the wire,
animations that repaint continuously, and lists that are hard to render. The
cross-language rules (a guard test per fix, numbers from a production build)
are in `practices/testing.md` section 10.

- Memoize expensive computations with `useMemo`. Don't memoize everything;
  React is fast enough for most renders.
- Every route page is a lazy chunk declared in `pageChunks.tsx`; the page the
  URL opens loads beside the bootstrap request and the rest preload once the
  first screen settles. The shell (`Layout`, the sidebar, the command
  palette) never statically imports a page, the rich-text editor, the
  topology canvas, voice or the shader; a dialog that carries the editor
  loads its form when it opens (`loadTicketForm`, `LazyCreateDocForm`).
  `pageChunks.test.tsx` fails when the shell's import graph picks one up
  again.
- Rows of a list that refetches or updates live (board cards and columns,
  chat messages and scroller items) are `memo` components fed stable props:
  callbacks from `useCallback` or `useLatestCallback`, never inline arrows,
  and derived arrays memoised by content, so one changed row renders one
  row. A refetch that returns new objects for unchanged rows re-renders the
  whole list otherwise.
- Inputs to a library's context are stable: dnd-kit sensor options are module
  constants and `SortableContext` `items` are memoised by content. A fresh
  array or options object rebuilds the context and re-renders every sortable
  under it.
- A custom property a library sets on `html` or `body` (the modal scroll
  lock's `--removed-body-scroll-bar-size`) is registered in `index.css` with
  `@property` and `inherits: false`; inherited, each set restyles every
  element on the page.
- Virtualize long lists (TanStack Virtual or react-window) when > 100 items.
- The React Flow canvas uses an external Zustand store + `useShallow` so only
  changed nodes re-render.
- No continuously-repainting animations (they peg GPUs on high-refresh
  displays); animate transform/opacity only.
- The React Compiler is not adopted. Do not add `babel-plugin-react-compiler`
  or its config without an ADR recording the decision. Independent of
  whether the compiler itself is ever adopted, `eslint-plugin-react-hooks`'s
  compiler-diagnostic rules (purity and memoization-safety checks merged
  into its `recommended` preset) are the target lint set.

---

## Quick reference: creating a new entity

Checklist for, e.g., a `Runner`:

1. Model: `models/Runner.tsx`, interface + Zod schema + inferred form type.
2. Enum (if any): `enums/Runner.tsx`, `as const` string map.
3. Hooks: `hooks/RunnerHooks.tsx`: `useFetchRunners`, `useFetchRunner(id)`,
   `useCreateRunner`, `useUpdateRunner`, `useDeleteRunner` (query keys exported).
4. Feed: `components/runner/RunnersFeed.tsx`, a hairline-row list (F1).
5. Dialogs: `CreateRunnerDialog.tsx`, `DeleteRunnerDialog.tsx`.
6. Pages: `pages/RunnersPage.tsx` + `pages/RunnerPage.tsx`.
7. Routes: add to `buildRoutes()` in `Router.tsx`.
8. Live: if the entity changes while someone looks at it, follow its topic
   ([The live topic contract](#the-live-topic-contract)).

## Quick reference: adding a new UI component

1. `bunx shadcn@latest add <name>` for primitives.
2. Custom components in `components/` or `components/<domain>/`.
3. Compose classes with `cn()`.
4. React-19 style: `ref` as a prop, no `forwardRef`.
5. Named exports for components and helpers.
6. If it renders on the canvas, theme it with the shared CSS variables.

---

## Lint & typecheck

The web workspace lives at `web/` in the monorepo (Bun). Before
committing, from `web/`:

```bash
bun run typecheck   # tsc --noEmit (TypeScript 7)
bun run lint        # eslint
bun run build       # vite build (catches what tsc/eslint miss)
```

Configured ESLint bases, from `web/eslint.config.js`: `eslint:recommended`
(flat), `typescript-eslint` recommended, `@tanstack/eslint-plugin-query`
flat/recommended, `eslint-plugin-react-hooks` recommended, plus a standalone
`react-refresh/only-export-components` warning. There is no
`react-refresh` recommended preset enabled, only that one rule. Enable
`verbatimModuleSyntax` in `tsconfig` so type-only imports are enforced
(matches the import rules above).

`tsc` is TypeScript 7, installed as `@typescript/native` and called by path
from the scripts. The plain `typescript` package stays on 6.0.x because
typescript-eslint needs the compiler API, which TypeScript 7 does not ship.
Dependabot ignores `typescript` past 6.0 in `web/` for that reason.
