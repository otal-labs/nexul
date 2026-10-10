# A project made in the wizard shows Continue setup until Finish

The project wizard creates the project at its Info step (ADR 0079), so leaving the wizard any time after that strands
a half-made project. The first fix remembered the stranded project in the browser that named it and showed a
dismissible banner on its board, which no other device and no other member ever saw.

Decision: the project records its own setup, on the server: whether the wizard's Finish was pressed, and each step
(`project`, `repository`, `service`, `env`, `reach`, `branches`) marked done or skipped, the exact stack the wizard created or adopted, and the detected environment key names.
Environment values stay on the stack; resuming reads them there, and the deploy uses that stack's build branch.
While setup is open, the project's section of the sidebar holds a single brand-filled Continue setup row instead of Board, Interview, Docs,
Memories and Settings, for every member who holds `projects:write` there; everyone else who can open the project sees
a muted "Being set up" line. Continue setup opens the wizard at the first step neither done nor skipped, including Environment when keys were detected, else at Done.

- **A signal, not a lock.** Only the sidebar changes. Direct links to the board, docs and settings still open, MCP and
  the HTTP gateway reach everything, and nothing is refused because setup is open.
- **Every step opens at any time.** Each node of the wizard's row is a button. Skip for now records the step skipped
  and moves to the next one instead of leaving the wizard; a step opened before its groundwork (Service with no
  repository) says what it needs and links there. Finish works with steps skipped, and Done lists them.
- **Only the wizard starts a project unfinished.** `POST /api/projects` and `project_create` take an optional
  `setup_finished`; omitted it is true, so every other caller still gets a ready project. Migration 0085 adds the
  columns with every existing project finished, so no sidebar changes on upgrade.
- **Finishing is not undone by revisiting.** Project Settings → General opens the wizard on a finished project to
  revisit a step, and the project stays finished. `PUT /api/projects/{id}/setup` and `project_update` can set
  `finished` back to false; the web app never does.
- **Continue saves before it moves.** Service creation leaves its summary available if the progress write fails;
  Continue retries the mark and stack identity without creating again. Another stack made later never changes
  which stack this setup resumes. Leaving through Info's Back or Done's actions also saves any pending service context;
  Finish writes that context together with finished, and a failed write keeps the session available for retry.
- **A scan describes one build source.** Once a service exists, its detected environment keys stay with that service;
  scanning another repository on a revisit neither replaces those keys nor changes the service's deploy branch.
- **An exact service belongs to the project.** Both adapters reject a missing or another-project stack before writing
  setup or its event. An empty stack id clears the selection.
- **A skip never undoes a done step**, so revisiting a step and skipping it keeps what it made.
- **Setup state belongs to the setup run.** The wizard reads and writes the recorded service only while the project is
  in setup or Settings reopened it to revisit. The Add a service door (`?add=1`) makes a new service and leaves the
  record, its step marks and its event alone, on a finished project and on one still in setup.
- **The service is for readers of stacks.** The service id and the environment key names reach HTTP and MCP callers who
  hold `stacks:read` on the project; anyone else gets the step marks and the finished flag.
- **Live.** `project.setup_changed` carries the finished flag and the step marks, and reaches whoever may open the
  project, so another member's sidebar flips the moment someone presses Finish. It names no service (a frame has one
  audience), so a client refetches the project to learn the service.

Rejected: keeping the browser-local reminder, which only the device that named the project ever saw. Also rejected:
deriving setup from what the project holds (a repository, a stack), because a project for docs only would then never
count as set up, and a skip is a choice the project has to remember.

The phone has no project wizard and no per-project page list, so it shows no setup state. Supersedes the
browser-local Continue setup banner. Decided 2026-10-10.
