# Containers reach the native server through host.docker.internal, and the server unit opens its port to Docker's bridges

Since ADR 0073 the server runs on the host, not on a Docker network, so a container the runner deploys cannot reach
it by a service name. The runner starts every container with `--add-host host.docker.internal:host-gateway`, and a
tunnel hostname routed without a local service forwards to `http://host.docker.internal:<server port>`. That is how
cloudflared, the first thing an instance deploys, reaches Nexul itself.

Traffic from a container to the host arrives on the host's INPUT chain, which many servers reject by default (cloud
images that only admit SSH). So on Linux the `nexul-server` unit inserts `iptables` rules accepting the server's port
from `docker0` and the `br+` bridges as it starts, and deletes them when it stops. Nothing is opened to other
interfaces. The rules live with the service rather than in saved firewall state, so a reboot or an uninstall leaves
nothing behind, and the install summary tells the owner the port was opened to containers.

Rejected: running cloudflared on the host network. It would reach `localhost` without a firewall change, but it could
no longer join application networks, so exposing another stack by its container name would stop working. Also
rejected: leaving the firewall alone and showing the owner the rule to add. The first deploy on a locked-down server
would then fail on a step the owner cannot see from the wizard.
