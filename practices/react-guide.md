# React practices

How the web client in `web/` is written and reviewed: domain folders, the
Feed, Dialog, and FormFragment taxonomy, grouped data hooks, co-located Zod
schemas, persisted Zustand. Where this file and a library's official
documentation disagree on an API, the documentation wins; this file is house
style on top of current APIs.

> Scope: the browser app only. The Go backend, the MCP server, and the runner
> are not covered here. See `README.md` for the product and `docs/adr/` for the
> decisions behind the backend.

---

## Table of contents

1. [Technology stack](#technology-stack)
2. [Nexul-specific rules (read first)](#nexul-specific-rules-read-first)
3. [Project structure](#project-structure)
4. [Naming conventions](#naming-conventions)
5. [Imports](#imports)
6. [Components](#components)
7. [Pages](#pages)
8. [Hooks](#hooks)
9. [State management (Zustand)](#state-management-zustand)
10. [Data fetching (TanStack Query)](#data-fetching-tanstack-query)
11. [Forms (React Hook Form + Zod)](#forms-react-hook-form--zod)
12. [API layer (REST over the HTTP gateway)](#api-layer-rest-over-the-http-gateway)
13. [Live events (WebSocket client)](#live-events-websocket-client)
14. [Topology canvas (React Flow)](#topology-canvas-react-flow)
15. [UI components (shadcn)](#ui-components-shadcn)
16. [Styling and theming (Tailwind)](#styling-and-theming-tailwind)
17. [Error handling](#error-handling)
18. [Routing (react-router)](#routing-react-router)
19. [Utilities](#utilities)
20. [Models & types](#models--types)
21. [TypeScript strictness](#typescript-strictness)
22. [Testing (frontend)](#testing-frontend)
23. [Accessibility](#accessibility)
24. [Performance](#performance)
25. [Anti-patterns to reject on review](#anti-patterns-to-reject-on-review)
26. [Quick reference: new entity](#quick-reference-creating-a-new-entity)
27. [Quick reference: new UI component](#quick-reference-adding-a-new-ui-component)
28. [Lint & typecheck](#lint--typecheck)

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
| Live events | native `WebSocket` client | Separate from Axios, see §13 |
| Notifications | Sonner | |
| Icons | lucide-react | |
| Async dialogs | react-confirm | Promise-returning dialogs via `useConfirmationDialog` / `useFormDialog` |

Versions are pinned in `web/package.json`; this table names the choice, not the version.

---

## Nexul-specific rules (read first)

These are the rules that are unique to this product and that override generic
React advice. They exist so the frontend stays a thin, correct peer of the
MCP-first backend.

1. **The browser talks to the HTTP/JSON gateway, never to MCP.** The backend
   exposes one domain/use-case layer through two adapters: an MCP server (for
   LLMs) and an HTTP/JSON gateway (for this app). The browser uses the gateway
   via Axios (§12). Do **not** speak JSON-RPC from the browser. This is
   ADR 0019.
2. **Live updates come over one WebSocket, not polling.** Runner status, deploy
   progress, and topology mutations arrive on a single WS connection (§13) and
   are fanned into Zustand / TanStack Query. Do not poll for status that the
   server already pushes.
3. **The topology canvas JSON is the infra model, for what it stores.** The
   stored JSON holds nodes, edges, and positions; that is what the backend
   stores and what the MCP server mutates (§14, ADR 0033). The dashed network
   boxes, gateway rows, and hostname pills rendered on screen are derived at
   render time from that JSON, not part of it. Do not maintain a parallel
   hand-written topology type; derive/validate the stored fields against the
   shared schema.
4. **Theming is shared between shadcn and React Flow.** Both read the same CSS
   variables (§16). A selected node uses the shadcn `ring`; the canvas
   background and edges respect light/dark automatically.
5. **Mirror the backend's string enums.** The Go backend serializes enums as
   strings. Frontend "enums" are string unions / `as const` objects (§20),
   never numeric TS enums.
6. **Domain folders, no barrels.** Group components by domain entity; import by
   explicit path; no `index.ts` re-export barrels (they defeat tree-shaking and
   hide dependency direction).
7. **`.tsx` everywhere is deliberate house style.** Even pure-TS model/enum
   files use `.tsx` in this repo. It is intentional, not a mistake. To stay
   clean under `verbatimModuleSyntax`, use `import type` for type-only imports
   so these files never emit unused-value-import errors.
8. **Third-party credentials go through the ticker, never a bare Save.** Any
   form that hands a token, secret, or app registration to an outside service
   (Cloudflare token, GitHub App, LiveKit keys) is a ticker: a list of named
   checks with a why under each, a Verify button that runs one request per
   check in parallel, and a Continue/Confirm that only unlocks once every row
   is green. Build it from `useTicker` (`hooks/useTicker.ts`) + `TickerRow`
   (`components/TickerRow.tsx`); the backend exposes the checks as
   `POST …/verify?check=<key>` returning 204 or the provider's error, and
   stores nothing. Examples: `ManualConnectorDialog`, `InstanceBootstrapPage`.
   Never collapse the checks into one request: a single red line can't tell
   the user which permission is missing.

---

## Frontend commandments (ENFORCED)

> These are **hard rules**, not suggestions. A PR that breaks them is not
> done, regardless of what the rest of the codebase currently looks like.
> **Do not copy a nearby file's pattern if it violates this section.** Much
> of the existing code predates these rules and has since been cleaned up.
> Copy the rule, not the drift.

### F1: component hierarchy, Page to Feed to Section to Card

Every screen decomposes into named, single-responsibility components:

- **`XxxPage`**: fetches, handles loading/error/empty, composes Feeds. No
  data transforms, no inline lists.
- **`XxxFeed`**: the list of entities (search/filter/actions). Lists are
  hairline rows, never a table library, see `practices/design-language.md`'s
  row pattern.
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
`data` never overlap).

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

`LoadingDisplay`, `ErrorDisplay`, `NoDataDisplay`, `Container`. **DO NOT**
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
  form state in React Hook Form. Never duplicate server state into
  `useState`/Zustand.

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
  api/                  # Axios instance + WS client + error types
  utils/                # Pure functions (TimeUtility.tsx and friends)
  Layout.tsx            # Root layout: header, <Outlet/>, footer
  main.tsx              # App entry
  Router.tsx            # createBrowserRouter definitions
```

Rules:

- One folder per domain entity under `components/`.
- No barrel/`index.ts` files, import by full path.
- All files use `.tsx` (house style; see rule 7 above).

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
| Page header | `PageHeader` | Semibold tracking-tight title + one-line subtitle (page-level marquee) |
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
| Ticker | `useTicker` + `TickerRow` | Named third-party checks that verify before Continue (rule 8) |
| Topology node | `XxxNode` | React Flow custom node (§14) |

---

## Imports

Order, top to bottom, blank line between groups:

1. Third-party libraries (`@tanstack/react-query`, `zod`, `lucide-react`, …)
2. shadcn/ui primitives (`@/components/ui/...`)
3. Local components (`@/components/...`)
4. Local hooks (`@/hooks/...`)
5. Local stores (`@/stores/...`)
6. Local models / enums (`@/models/...`, `@/enums/...`)
7. Local utilities (`@/lib/utils`, `@/utils/...`)

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
`ServicesFeed.tsx`. `practices/design-language.md`'s "List / row" pattern spec
is the row layout of record; this section covers the component shape only.

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

## Pages

Every page follows the explicit `isPending / error / empty / data` pattern, in
that defensive order, rendered with `&&` blocks inside the page's `Container`,
never `if (x) return <Component/>` early returns (see commandment F2). (React
19's `use` + `Suspense` is available, but we keep the explicit pattern for
clarity and uniform error UIs.)

```tsx
import { useFetchTickets } from "@/hooks/TicketHooks";
import { TicketsFeed } from "@/components/ticket/TicketsFeed";
import { Container } from "@/components/Container";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { NoDataDisplay } from "@/components/NoDataDisplay";

export const TicketsPage = () => {
  const { data, error, isPending } = useFetchTickets();
  return (
    <Container>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {data && data.length === 0 && <NoDataDisplay message="No tickets yet" />}
      {data && data.length > 0 && <TicketsFeed tickets={data} />}
    </Container>
  );
};
```

Detail page:

```tsx
import { useParams } from "react-router";
import { useFetchTicket } from "@/hooks/TicketHooks";

export const TicketPage = () => {
  const { ticketId } = useParams<{ ticketId: string }>();
  const { data: ticket, error, isPending } = useFetchTicket(ticketId!);
  return (
    <Container>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {ticket && <TicketDetail ticket={ticket} />}
    </Container>
  );
};
```

---

## Hooks

Group all hooks for an entity in one file (`TicketHooks.tsx`). Export query
keys as constants. Always invalidate on success; always toast on success/error.

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

Hook rules:

- Query keys are exported constants (`getXxxKey`).
- Parameterized keys include the param: `["getTicket", id]`.
- Invalidate the relevant list key in `onSuccess`.
- `toast.success` on success, `toast.error(errorMessage(error))` on error.
- Parse field errors via the shared `errorMessage()` (§17).

---

## State management (Zustand)

### The three tiers of state

| Tier | Tool | What lives here |
|---|---|---|
| Server state | TanStack Query | Anything fetched from the API |
| Client UI state | Zustand | Theme, sidebar open/closed, canvas viewport |
| Form state | React Hook Form | In-progress form values |

Do not duplicate server state into Zustand. Do not put form state in Zustand.
Do not fetch in `useEffect` + `useState`; use TanStack Query (see F5).

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

## Data fetching (TanStack Query)

| Operation | Hook | Method |
|---|---|---|
| List | `useFetchXxx` | GET list |
| One | `useFetchXxx(id)` | GET one, `enabled: !!id` |
| Create | `useCreateXxx` | POST |
| Update | `useUpdateXxx(id)` | PUT/PATCH |
| Delete | `useDeleteXxx` | DELETE |

Invalidation strategy: after a mutation, invalidate the list key; optionally
refetch the single resource. The WS client (§13) may also invalidate keys when
the server pushes a change, so the UI stays live without polling.

Data-fetching rules:

- One file per domain: `hooks/TicketHooks.tsx`, `hooks/DocHooks.tsx`.
- Export query keys as constants (`getXxxKey`).
- Every mutation invalidates the relevant list key in `onSuccess`, unless
  the response is the full entity and the list is high-churn (chat
  messages): then `setQueriesData` patches it in place, with an optimistic
  row from `onMutate` where the user expects instant feedback.
- The WS client invalidates keys on push, **do not poll**.
- Use `enabled: !!id` for detail queries so they don't fetch with an
  undefined id.
- Do **not** default list data to `= []` when you render a length-based
  empty state (see the "React Query: don't over-guard" callout in F2).
- A query used in more than one place (a component plus a prefetch, or two
  hooks reading the same resource) is defined once with `queryOptions()` and
  shared between them. One key and one fetcher in one place means the two
  call sites cannot drift out of sync with each other.

---

## Forms (React Hook Form + Zod)

Schemas live next to the model they validate; form types are always
`z.infer<typeof Schema>`, never duplicated by hand. Forms go through React
Hook Form and Zod rather than React 19's form Actions and `useActionState`
so validation stays typed end to end and field errors map onto the shared
error envelope (§12) instead of a plain action-state string. Zod is the
schema library; which major version of it the repo runs is a decision for
the whole codebase, never a per-file choice.

```tsx
import { z } from "zod";

export interface Ticket {
  id: string;
  title: string;
  body: string;
}

export const SaveTicketFormSchema = z.object({
  title: z.string().min(1, "Title is required"),
  body: z.string().min(5, "Body must be at least 5 characters"),
});

export type SaveTicketFormData = z.infer<typeof SaveTicketFormSchema>;
```

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

- Co-locate the Zod schema with the model interface.
- Use `FormInput` / `FormSwitch` for standard fields; raw `FormField` for the
  rest.
- Compose big forms with `useFormContext` fragments.
- Disable submit with `form.formState.isSubmitting`.
- Surface server-side root errors via `setError("root", ...)` in `onError`.

---

## API layer (REST over the HTTP gateway)

The browser uses **one** Axios instance that points at the Go HTTP/JSON
gateway. The gateway and the MCP server are two adapters over the same
use-cases (ADR 0019); the browser never touches MCP. Axios is the choice
here, not a thin `fetch` wrapper, because `client.tsx` leans on its
request/response interceptors for the two things this app actually needs:
bearer-token injection on every request and a single place to catch a global
401. A thinner client would need the same hooks wired up by hand for no
gain at this app's scale.

```tsx
import axios from "axios";
import { useSessionStore } from "@/stores/sessionStore";

export interface ApiErrorBody {
  message: string;
  code: string;
  errors?: Record<string, string[]>;
}

export const api = axios.create({ baseURL: import.meta.env.VITE_API_URL });

api.interceptors.request.use((config) => {
  const token = useSessionStore.getState().token;
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

api.interceptors.response.use(
  (response) => response,
  (error: AxiosError<ApiErrorBody>) => {
    if (error.response?.status === 401) {
      useSessionStore.getState().logout();
      redirectToLogin();
    }
    return Promise.reject(error);
  },
);
```

`api` is a plain Axios instance; it does not unwrap `.data` for you. Every
call site reads `.data` off the response itself.

The `ApiErrorBody` shape **matches the backend error envelope** (`internal/platform/httpx`): a
`message`, a stable machine `code`, and optional per-field `errors`. The
frontend parser in §17 reads exactly this.

> Dev mode: the gateway serves the SPA same-origin in production (embedded
> webui) and sends no CORS headers, so `vite.config.ts` proxies `/api`,
> `/auth`, and `/ws` to `http://localhost:8080`. Run the dev server with
> `VITE_API_URL=/`.

Typed usage: type the response with generics, then read `.data` off the
promise with `.then((r) => r.data)`:

```tsx
const tickets = await api.get<Ticket[]>("/api/tickets").then((r) => r.data);
const id = await api.post<string>("/api/tickets", input).then((r) => r.data);
```

API rules:

- Always use `api`, never raw `axios` (except one-off OAuth redirects).
- Always type responses with generics.
- Bearer token is injected by the interceptor; read it with `getState()`.
- Handle errors at the hook/component layer, not in the interceptor. The
  interceptor only handles a 401: logout, then redirect.

---

## Live events (WebSocket client)

Status that the server pushes (runner heartbeats, deploy progress, topology
mutations, dead-letter alerts) arrives on **one** WebSocket. This is separate
from Axios and is the consumer side of the runner WebSocket protocol (ADR 0031).

Design:

- One connection, owned by a provider mounted once at the layout root.
- Exponential-backoff reconnect with jitter; pause when the tab is hidden.
- Incoming frames are typed (`{ topic, type, payload }`) and dispatched:
  - topology/status frames patch the React Flow store / status store;
  - domain-change frames call `queryClient.invalidateQueries(...)` for the
    affected key, so TanStack refetches the canonical state;
  - a frame carrying the whole entity (chat messages) patches the cached
    list with `setQueriesData` instead, so a busy thread never refetches.
- The WS client **never** mutates server state, it is read-side only. Writes
  go through the REST gateway.

Sketch (kept dependency-free):

```tsx
import { useEffect, useRef } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useFlowStore } from "@/stores/flowStore";

export const useLiveEvents = (url: string) => {
  const client = useQueryClient();
  const attempt = useRef(0);

  useEffect(() => {
    let ws: WebSocket;
    let timer: ReturnType<typeof setTimeout>;

    const connect = () => {
      ws = new WebSocket(url);
      ws.onopen = () => (attempt.current = 0);
      ws.onmessage = (ev) => {
        const frame = JSON.parse(ev.data) as ServerFrame;
        dispatch(frame);
      };
      ws.onclose = () => {
        const delay = Math.min(30_000, 500 * 2 ** attempt.current++) + Math.random() * 250;
        timer = setTimeout(connect, delay);
      };
    };

    const dispatch = (frame: ServerFrame) => {
      if (frame.topic === "topology") useFlowStore.getState().applyServerPatch(frame.payload);
      else client.invalidateQueries({ queryKey: [frame.topic] });
    };

    connect();
    return () => {
      clearTimeout(timer);
      ws?.close();
    };
  }, [url, client]);
};
```

Rules:

- Mount once (layout root). Components subscribe to *stores*, not to the
  socket.
- Treat WS data as a hint to refresh / patch; the REST gateway remains the
  source of truth on read.
- Carry a `trace_id` on frames where present; log it via the same structured
  logger convention as the backend.

---

## Topology canvas (React Flow)

The deploy topology is rendered with `@xyflow/react`. Visual tokens (canvas
field, node cards, edges, selection ring) live in the shared theme; see
`practices/design-language.md` (the spec of record) and `web/src/index.css`
(the token source of record); this section covers mechanics only. The nodes
are ours and carry Nexul-specific signals (MCP index health, owning
ticket, live deploy status).

### The canvas store (external Zustand + `useShallow`)

React Flow performs best when node/edge state lives in an external store and
components select slices with `useShallow`. This store also owns the
server-pushed patches from §13.

```tsx
import { create } from "zustand";
import {
  addEdge,
  applyEdgeChanges,
  applyNodeChanges,
  type Connection,
  type Edge,
  type EdgeChange,
  type Node,
  type NodeChange,
} from "@xyflow/react";
import type { ServiceNodeData } from "@/components/topology/ServiceNode";

export type ServiceNode = Node<ServiceNodeData, "service">;

export type FlowStore = {
  nodes: ServiceNode[];
  edges: Edge[];
  onNodesChange: (changes: NodeChange[]) => void;
  onEdgesChange: (changes: EdgeChange[]) => void;
  onConnect: (c: Connection) => void;
  applyServerPatch: (patch: TopologyPatch) => void;
};

export const useFlowStore = create<FlowStore>((set) => ({
  nodes: [],
  edges: [],
  onNodesChange: (changes) => set((s) => ({ nodes: applyNodeChanges(changes, s.nodes) })),
  onEdgesChange: (changes) => set((s) => ({ edges: applyEdgeChanges(changes, s.edges) })),
  onConnect: (c) => set((s) => ({ edges: addEdge(c, s.edges) })),
  applyServerPatch: (patch) => set((s) => mergePatch(s, patch)),
}));
```

The app wires it with a shallow selector so only changed slices re-render:

```tsx
import { useShallow } from "zustand/react/shallow";
import { ReactFlow, ReactFlowProvider, Background, BackgroundVariant } from "@xyflow/react";

const nodeTypes = { service: ServiceNode };
const edgeTypes = { relation: RelationEdge };

const Flow = () => {
  const { nodes, edges, onNodesChange, onEdgesChange, onConnect } = useFlowStore(
    useShallow((s) => ({
      nodes: s.nodes,
      edges: s.edges,
      onNodesChange: s.onNodesChange,
      onEdgesChange: s.onEdgesChange,
      onConnect: s.onConnect,
    })),
  );
  return (
    <ReactFlow
      nodes={nodes}
      edges={edges}
      nodeTypes={nodeTypes}
      edgeTypes={edgeTypes}
      onNodesChange={onNodesChange}
      onEdgesChange={onEdgesChange}
      onConnect={onConnect}
      fitView
    >
      <Background variant={BackgroundVariant.Dots} />
    </ReactFlow>
  );
};

export const TopologyCanvas = () => (
  <ReactFlowProvider>
    <Flow />
  </ReactFlowProvider>
);
```

### Custom node: one flexible `ServiceNode`

A single component covers every card; the icon and footer slots are driven by
`data`, not by per-type components.

```tsx
import { Handle, Position, type NodeProps } from "@xyflow/react";
import { StatusBadge } from "@/components/topology/StatusBadge";
import type { ServiceNodeData } from "./ServiceNode";

export const ServiceNode = ({ data }: NodeProps<ServiceNodeData>) => (
  <div className="w-64 rounded-lg border bg-card p-3 shadow ring-0 data-[selected=true]:ring-2">
    <Handle type="target" position={Position.Left} className="!bg-muted-foreground" />
    <div className="flex items-center gap-2">
      <RuntimeIcon runtime={data.runtime} />
      <span className="font-semibold">{data.name}</span>
    </div>
    {data.url && <a className="text-xs text-muted-foreground" href={data.url}>{data.url}</a>}
    <StatusBadge status={data.status} />
    {data.replicas != null && <Footer label={`${data.replicas} replicas`} />}
    {data.volume && <Footer label={data.volume} />}
    <Handle type="source" position={Position.Right} className="!bg-muted-foreground" />
  </div>
);
```

### Typed edges carry meaning

Edges are typed (`depends_on`, `connects_to`, `mounts`) because that topology
is exactly what the MCP server answers with ("what breaks if I redeploy
postgres?"). Render them dashed with `smoothstep`:

```tsx
import { BaseEdge, getSmoothStepPath, type EdgeProps } from "@xyflow/react";

export const RelationEdge = (props: EdgeProps) => {
  const [path] = getSmoothStepPath(props);
  return <BaseEdge path={path} style={{ strokeDasharray: "4 4" }} />;
};
```

### Canvas JSON is the infra model, for what it stores (ADR 0033)

`getNodes()` / `getEdges()` serialized holds nodes, edges, and positions;
that is the topology the backend stores and the MCP server mutates. Validate
it against the shared schema on load and save; do not keep a second
hand-written topology type. The dashed network boxes, gateway rows, and
hostname pills rendered on the canvas are derived at render time from that
stored data, not part of the serialized JSON itself. First paint uses
`elkjs` for dependency-aware auto-layout; manual drags persist as positions.

### Theming the canvas

Override React Flow's CSS with the same variables as shadcn so nodes, edges,
and the minimap follow light/dark for free; selected node = shadcn `ring`.
Node detail (logs, env, deploy history, SSH/docker config) opens in a shadcn
`Sheet` on click. Canvas is overview, sheet is depth.

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
- Icons from `lucide-react`: `<Plus className="h-4 w-4" />`.
- Named-export custom components; shadcn primitives follow whatever the CLI
  emits (it now generates React-19-compatible code).
- Enter/exit motion uses `tw-animate-css` utilities (`animate-in`/`animate-out`
  + `fade-in-0`, `zoom-in-95`, `slide-in-from-*`) with `ease-standard` and
  150-250ms durations; the global `prefers-reduced-motion` block in
  `index.css` disables all transitions/animations.

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

Tailwind v4 is configured in CSS, not a JS config. shadcn's v4 setup uses
`@theme inline` to expose CSS variables to utilities, plus a class-based dark
variant. **Tokens are not specified here, see
`practices/design-language.md` (the visual spec of record) for the direction
and `web/src/index.css` (the token source of record) for the values.**
Structurally the theme adds, beyond the default shadcn set: a `surface-2`
token (board fields, canvas, section breaks), the status tokens (`--success`,
`--warning`, `--info`), an elevation scale
(`shadow-card`/`-elevated`/`-overlay`), the motion tokens (`ease-standard` =
`cubic-bezier(0.25, 0.1, 0.25, 1)`, `ease-out` =
`cubic-bezier(0.16, 1, 0.3, 1)`), and the locally bundled type stack: Inter
Variable (UI + display, tight tracking) + JetBrains Mono (technical data),
via `@fontsource-variable`, no CDN. No serif or script type.

Radius system: **6px interactive** (`rounded-md`), 8px large cards
(`rounded-lg`), pills (`rounded-full`) only for chips/badges/avatars/status,
never buttons or inputs.

Use container queries (`@container`, `@min-*`/`@max-*`) for component-level
responsiveness, not viewport breakpoints, whenever a component's layout
should react to the space it is actually given (a card in a narrow sidebar
versus the same card in a wide panel). The mobile-first law is that a
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
  surfaces. Status hues come from `success` / `warning` / `info` /
  `destructive` tokens, rendered as a colored icon or dot next to plain text,
  never a filled or tinted-background chip, except the board card's own pill
  row (see `practices/design-language.md`).
- Conditional classes via `cn()`:

```tsx
<div className={cn("rounded p-4", active && "bg-accent", className)} />
```

- Wrap page content in `Container` (`mx-auto w-full max-w-7xl`).
- The React Flow canvas reads these same variables (§14): one theme, two
  surfaces.

---

## Error handling

Three levels, plus a shared parser that matches the backend envelope (§12).

Shared parser:

```ts
import type { ApiErrorBody } from "@/api/client";
import type { AxiosError } from "axios";

export const errorMessage = (error: unknown): string => {
  const e = error as AxiosError<ApiErrorBody>;
  const body = e?.response?.data;
  const firstField = body?.errors ? Object.values(body.errors)[0]?.[0] : undefined;
  return firstField || body?.message || e?.message || "Something went wrong";
};
```

1. **API**: the gateway returns `{ message, code, errors? }`; the parser above
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
- The shell is a **sidebar layout**: a sticky left rail (`Layout.tsx`) with
  grouped nav (Work / Deploy / Manage via `SidebarNavGroup`), a collapse
  toggle (instant width swap, no layout animation, see the motion rules), and
  ThemeToggle + NotificationBell + Sign out pinned at the bottom. Pages
  render inside `<main>` under `Container` (`mx-auto w-full max-w-7xl`).
- Catch-all `*` renders `ErrorPage`, last.
- Auth-gated routes added conditionally.

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
these `.tsx` (rule 7); use `import type` so they stay clean.

---

## Models & types

Model files pair an interface with its Zod schema and the inferred form type:

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
  exist before the mock factory runs. See `TopologyHooks.test.tsx` for the
  full pattern.

### What NOT to test

- shadcn primitives (they have their own tests upstream).
- That `cn()` merges classes correctly (it's `twMerge(clsx(...))`).
- Snapshot tests for layout (break on every CSS change, teach nothing).

### Coverage config

Thresholds and the exclude list live in `web/vitest.config.ts`; read that
file directly; a copy printed here would drift from the real config.

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

- Memoize expensive computations with `useMemo`. Don't memoize everything;
  React is fast enough for most renders.
- Use `React.lazy` + `Suspense` for route-level code splitting.
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

## Anti-patterns to reject on review

The short reject-list; the Frontend Commandments (F1-F7) are the authority.

1. **Inline `.map()` rendering a `<section>`/large JSX block**: extract a
   named component (F1).
2. **`if (x) return <Component/>` for rendering**: use `&&` blocks (F2).
3. **Loading/error/empty checks buried at the bottom**: order them
   negative-first (F2).
4. **Hand-rolled `<p>Loading…</p>` / inline empty states**: use the shared
   displays (F3).
5. **Ternaries choosing between components**: split into `&&` blocks (F4).
6. **Redundant `!isPending && !error &&` guards / `= []` defaults feeding an
   empty state / bare `<>` fragments**: React Query overkill (F2).
7. **Fetching or derived values in `useEffect`/`useState`**: TanStack Query
   + `useMemo` (F5).
8. **Prop drilling beyond 2 levels**: fetch in the consumer or use a store
   (F6).
9. **Helper components defined inside a page file**: promote to
   `components/<domain>/` (F7).
10. **Raw API calls in pages/components**: all calls go through typed hooks
    in `hooks/XxxHooks.tsx`.
11. **Hard-coded palette classes** (`bg-yellow-100`, `text-blue-800`) on
    themed surfaces: use semantic tokens.

**Copying existing code that does any of the above is not a defense.** Much of
the codebase predates a given rule and was fixed in a later cleanup pass.
Implement against the rule, not the drift.

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
8. WS: if the entity has live status, ensure its topic invalidates the right
   query key in the live-events dispatcher (§13).

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
bun run typecheck   # tsc --noEmit
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
