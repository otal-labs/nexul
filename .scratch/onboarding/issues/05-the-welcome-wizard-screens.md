# 05: How the Welcome wizard, the account menu entry and its dot look

Type: prototype
Status: resolved
Blocked by: 02

## Question

Build the screens behind a live picker in auto design mode, on the existing
design language, at 768, 1024 and 1440px:

- The wizard frame: the step list, skip, done, and how it ends.
- Each step:
  - Profile, replacing today's first-login step.
  - Connect GitHub.
  - Your computer: the Add a computer flow with its three live checks, then
    setup.
  - Your projects, including the "not in a project yet" state.
  - Add a deploy server: the command, then waiting for the server to connect.
  - Get the phone app.
- The owner wizard handing over to it.
- The account menu: "Welcome" at the top with what's left, and the red dot on
  the trigger, reusing `UpdateDot`'s shape in the status hue.

## Answer

Decided in auto design mode, 2026-10-13. The prototype is on the pushed branch
`proto/welcome-wizard`, at `/wizard/welcome` and never merged. The locked references are in
`~/Code/Images/onboarding/05-welcome-wizard/`.

**Frame: Rail (picked).** It scored 6.5 of 7. The runner-up, Row, scored 4.5, and Hub scored 3.5.
- It sits on `ShowcaseSurface` with no sidebar, inside `max-w-6xl` with `px-6 lg:px-10`.
- Top bar: the logo, and a ghost "Finish later". Closing the wizard this way counts as its
  first run.
- Left column (13rem, or 15rem from a 56rem container): a `WELCOME` microheader and a mono
  "N of M done", then one 20px node per step with the project wizard's states.
  - Done: a `success` check.
  - Skipped: a dashed ring.
  - To do: a hollow ring.
  - Current: a `brand` ring with a dot.
  - Done and skipped steps carry an 11px mono label. Every step is clickable. A hairline
    `border-r` separates the column.
- Right column: `max-w-3xl`, with a Fraunces title at 2rem and a muted lead.
- Footer: Back, a ghost "Skip for now", and one primary. Skip is hidden once a step is done. A
  step that was already done shows a `bg-card` row saying what it found ("Connected as @…"),
  and only Continue.
- Motion uses existing values only: the step slides 8px over 180ms `--ease-out`, with
  reduced motion respected. Checks `pop-in`, and the finish view rises over 200ms.

**Steps**
- **Profile**: avatar (Upload and Remove) and display name. No Skip: the sign-in already
  filled it, so Continue is the skip.
- **Connect GitHub**: three numbered lines saying what it gives (pick repositories, see your
  pull requests, read-only), and a primary "Connect GitHub" with the GitHub mark.
- **Your computer**: the Add a computer step as it is. The command sits beside the connection
  panel, with Checks (Computer connected, T3 Code found, Paired) side by side from 1440 and
  stacked below. "Set up" stays disabled until Paired, then the setup rows show, and Continue
  waits for setup to be confirmed.
- **Your projects**: one hairline row per project (a 32px prefix tile, name, a mono access
  label, a muted line of what they can do there, a ghost "Open").
  - The access labels follow whatever the permission rework names.
  - When the person is in none: `EmptyState` saying "Your team hasn't added you to a project
    yet. Ask whoever invited you. This step ticks itself the moment they do.", with Skip as the
    only way on.
- **Add a deploy server**: the runner command snippet and when it expires, then a "Server
  connected" `CheckRow` that waits, then reads "build-01 · linux amd64 · added by you".
  Continue waits for the server.
- **Get the phone app**: the existing `QrFrame`, numbered "Install the app" (with a link) and
  "Scan this code" with a mono countdown, then a "Phone connected" `CheckRow`. This settles the
  map's phone-step question: the wizard shows the code itself.
- **End**: a Fraunces "N things left for later", or "You're set up". One row per step that
  isn't done (node, name, mono Skipped or To do, ghost Open), then "Go to <workspace>".
  - Copy fix before building: the line under the title says only steps still to do show the
    red dot, since skipped steps don't light it.

**Owner hand-over**: the owner wizard's "Set up T3 Code" step becomes a centred
`WizardConfirmation` reading "Workspace ready" and "Next, what's yours to set up: GitHub and
your computer". The Welcome wizard opens on the first step that isn't done, and ends with
"Create your first project", which leads into the project wizard.

**Account menu**
- The dot is `UpdateDot`'s shape in `bg-destructive`, on the avatar's top-right corner
  (`-top-0.5 -right-0.5`, `ring-2 ring-background`), in the full sidebar and the icon rail.
  The trigger's label gains ", N welcome steps left".
- The popover widens from `w-56` to `w-64`. Its first row is a sparkles icon, "Welcome", a 6px
  destructive dot and a mono "N left". When everything is done it reads "6/6" and stays, so
  the wizard is always reachable.
- Under it sits one indented row per step that isn't done, with a muted "Skipped" where it
  applies. Each row opens the wizard on that step.

**Runner-up, Row**: the project wizard's horizontal node row above a `max-w-xl` column. Every
step body, the footer and the end view are shared, so swapping means changing only the frame
(`RowFrame` in `web/src/prototype/welcome/frames.tsx` on the prototype branch).

**Not verified**: motion recordings, real pushed updates, other palettes, keyboard focus order,
and more than six steps.
