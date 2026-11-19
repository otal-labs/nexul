# Choose which roles Compare roles shows

**Status:** ready-for-agent
**Blocked by:** None — can start immediately

## What

Compare roles (Access → Roles and people) shows every role but the Owner as a matrix column, so comparing two roles that are not side by side (Client and Billing) means scrolling sideways past the ones between them.

Add a "Roles shown" menu in the matrix head: one checkbox per role, all on by default, remembered per browser (a per-viewer convenience, so browser storage). The "+N" control and the end fade keep counting only the columns that are shown.

## Acceptance

- Unticking a role removes its column; ticking it again puts it back in editor order.
- With two roles ticked, both columns fit without scrolling at 768, 1024 and 1440px.
- The choice survives a reload in the same browser and never reaches the server.
- `practices/design-language.md` "Role matrix" and the guide page `people-and-access.md` describe the menu.

## Comments
