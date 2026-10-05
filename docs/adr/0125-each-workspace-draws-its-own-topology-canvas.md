# Each workspace draws its own topology canvas

The topology canvas was one instance-wide map (ADR 0087, ADR 0089), so `/<slug>/topology` drew every workspace's
stacks in every workspace, and anyone holding `topology:read` anywhere read the service names and addresses of
workspaces they were never invited to. The sidebar and the URL put the canvas inside a workspace while the server
kept it outside every one.

Decision: a workspace has its own canvas.

- **What shows.** A workspace's canvas shows the containers of its projects' stacks, and an instance stack (a
  gateway) only while one of its exposures routes a hostname to one of those containers. A gateway serving two
  workspaces shows in both, listing in each only the hostnames that reach that workspace's services. An instance
  stack that routes into no workspace, such as the tunnel in front of Nexul itself, shows on no canvas; Settings →
  DNS and its stack page still list it.
- **Storage.** The `topology` row keyed `default` becomes the service registry: one node per container, kept by the
  deploy consumers with its name, status, and address, never read as a canvas by anyone. Each workspace's row is
  keyed by its id and holds what its people chose: positions, drawn nodes and edges, and the camera. A read composes
  the two, so a status change reaches every canvas showing the service without being written into each. Migration
  0074 copies the shared canvas to the first workspace, so its layout survives; the others lay their services out on
  first view.
- **Access.** Reads and writes check the canvas's workspace (`Require`, not `RequireAnywhere`), so an outsider gets
  not found and a member without `topology:read` gets forbidden. Topology stays an instance-area permission in the
  table, which keeps a Restricted member (ADR 0097) off a canvas that lists every project's services.
- **Adapters.** `/api/topology` takes `?workspace=`; a client that names none gets the first workspace, so a released
  client keeps working (ADR 0082). `topology_get` and `topology_update` take a required `workspace_id` in place of
  `environment`, and the resource is `topology://{id}` with a workspace id. `topology.updated` carries
  `workspace_id` for a canvas change and reaches that workspace's automations; a registry change names no workspace
  and reaches none, while the live socket pushes each workspace its own composed canvas.

Rejected: filtering the one shared canvas by the URL's workspace. The drawn nodes and edges would have no owner and
would still show everywhere, and the gate would still answer to the most generous workspace.

The trade-off: a hand-drawn node or edge from before the split lives only on the first workspace's canvas, and a
registry change composes and pushes one canvas per workspace, which is cheap at the size of one team's instance.

Amends ADR 0033 (each workspace stores its own map), ADR 0087 and ADR 0089 (the topology is no longer instance-level
on the server).

Decided 2026-10-05.
