package plays

// InvestigateCrashKey is the built-in key of the stack play whose seeded auto play runs it when a service crashes
// (ADR 0156).
const InvestigateCrashKey = "investigate-crash"

const investigateCrashLabel = "Investigate a crash"

const investigateCrashDescription = "Reads a crashed service's logs and recent deploys, finds the cause, and writes it down. It never deploys."

const investigateCrashInstructions = "A service in a stack crashed. Find out why and write it down. The stack is named below, and when a crash started this run so are the service and its outage. " +
	"Read the stack with `stack_get`: its services, which of them are down now with their exit code and restart count, and its recent deploys. Pass `logs` with a service's name to read that container's last lines; the cause is usually in the output just before the exit. When no service is named below, read the logs of each service that is down. " +
	"Then look at the recent deploys for what changed shortly before the crash, and find the cause in the project checkout you are running in: the commit, the config or the dependency that explains the exit code (137 with `oom` set is the container running out of memory). " +
	"Write your findings where the team will see them. Look for an open ticket in the stack's project that already names this crash, with `ticket_list` and a `query` of the service name, and add your findings to its thread with `message_post`; when there is none, create one with `ticket_create`, titled with the service and what happened, with the findings in its body. " +
	"State the cause, or your best-supported guess when you cannot be sure, the evidence for it (log lines, the deploy, the commit), and what you would change. Say plainly when you could not find the cause. " +
	"Never deploy, restart, stop or edit the stack or any of its services: this play only reads and reports. Open a fix pull request only if the instructions for this run say so. " +
	"If the evidence points at Nexul itself, such as the runner, the deploy engine or how the stack was built, rather than at the application, say so in your findings and tell the person to triage it with the `nexul-triage` skill. " +
	"Reply with the cause in one sentence and where you wrote it."

// investigateCrashAutoPlay is the seeded auto play: a crashed service starts the play on the computer a person picks,
// off until one is picked and it is switched on.
func investigateCrashAutoPlay() *AutoPlayInput {
	return &AutoPlayInput{
		Moment: MomentStackServiceCrashed, RunOn: RunOnComputer,
		Conditions: Conditions{Match: MatchAll, Groups: []Group{}},
		Priority:   Priority{Rules: []PriorityRule{}, Otherwise: LevelNormal},
	}
}
