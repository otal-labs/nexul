# Deploying is a choice in the project wizard, and the wizard re-reads the attached repository

A project set up with its repository attached but not deployed could not be deployed through the wizard later. The
scan that finds a Dockerfile or compose file lived only in the browser that picked the repository, so after a reload,
on another device, or on a revisit from Settings, Service asked to "scan it again" and sent the person to Repository,
whose Continue kept the repository and went back to Done. The only way out was to change the repository to itself.

Decision: the scan is derived, not remembered. Whenever the wizard reaches Deploy or Service with no service and no
scan, it scans the project's attached repository that no service builds from, on whatever device it runs. Deploying is
its own step, Deploy, between Repository and Service in the setup run: it shows what the scan found and asks Deploy now
or Later. Later records the deploy steps skipped and lands on Done, which offers Deploy now until a service exists.
Picking a repository in the setup run attaches it without scanning; the scan belongs to Deploy.

- **Deploy is not a setup step on the server.** The recorded steps stay `project`, `people`, `repository`, `service`,
  `env`, `reach`, `branches` (ADR 0143). Deploy reads as done once Service is, and as later while Service is skipped,
  and Continue setup opens Deploy wherever it would have opened Service.
- **The Add a service and Attach doors keep scanning on pick.** They exist to deploy, so they have no Later and no
  Deploy step.

Rejected: storing the scan with the project, which keeps a copy that goes stale the moment the repository changes,
when a scan is one request away. Also rejected: keeping "Attach without deploying" on the repository row, where the
choice to deploy later was a ghost button nobody read as a decision.

Amends ADR 0143, whose Service step asked for a rescan. Decided 2026-10-20.
