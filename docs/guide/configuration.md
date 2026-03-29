# Guide: Configuration

A `.lorah` directory contains the files that define a unit of work for the agent loop. It sits at the project root by default (overridable with `--dir`).

```
.lorah/
├── plan.md          # scope and acceptance criteria
├── prompt.md        # agent instructions for each iteration
└── settings.json    # Claude Code CLI settings
```

## Plan file

The plan file is the output of the scoping step — [Phase 1](workflow.md#phase-1-scope-the-work) in the spec-driven workflow. It defines what is being built and what done looks like. The agent loop uses it as the contract between the human and the agents.

A plan file contains:

- **Scope** — what is being built, at the level of a brief description and a list of capabilities. Reference the design specs rather than duplicating them.
- **Boundaries** — constraints and invariants that apply across the work (e.g., "stdlib only", "no external dependencies").
- **Acceptance criteria** — concrete, verifiable conditions that define when the work is complete. An agent should be able to check each criterion against git state and test results.

A plan file does not contain individual tasks. Task selection happens inside the loop, where each agent picks the next task based on current state.

## Prompt file

The prompt file is a markdown file piped to Claude Code on each loop iteration. It defines the agent's role, workflow, and constraints. This is the primary lever for controlling agent behavior.

A prompt file typically contains:

- **Role** — what the agent is and what it does in one sentence.
- **Workflow steps** — the sequence the agent follows each iteration: orient (check git history), select (pick next task), execute, verify, commit, exit.
- **Rules** — hard constraints the agent must follow (e.g., one task per invocation, strict TDD boundary).
- **Blocked workflow** — what to do when the agent encounters an issue it cannot resolve.

The prompt does not need to be elaborate. Agents are capable of self-managing within clear constraints. Focus on boundaries and invariants rather than detailed instructions for every scenario.

The prompt references the plan file for scope and the design specs for behavioral details. It should not duplicate either.

## Settings

`settings.json` is a standard Claude Code CLI settings file. Pass it via the `--settings` flag:

```sh
lorah run prompt.md --settings .lorah/settings.json
```

Common fields:

```json
{
  "model": "sonnet",
  "permissions": {
    "defaultMode": "bypassPermissions"
  },
  "sandbox": {
    "enabled": true,
    "autoAllowBashIfSandboxed": true
  },
  "includeCoAuthoredBy": false
}
```

- **model** — which Claude model to use.
- **permissions** — `bypassPermissions` is typical for autonomous loops where no human is approving each action.
- **sandbox** — enables sandboxed execution. `autoAllowBashIfSandboxed` avoids permission prompts for shell commands when sandboxing is on.

See the [Claude Code documentation](https://docs.anthropic.com/en/docs/claude-code) for all available settings.

## Claude flags

Additional Claude CLI flags can be passed after the prompt file:

```sh
lorah run prompt.md --settings .lorah/settings.json --model claude-opus-4-6 --max-turns 50
```

Flags are passed through to the `claude` CLI unchanged. Common flags:

- `--settings <file>` — path to settings file
- `--model <model>` — override the model (takes precedence over settings.json)
- `--max-turns <n>` — limit the number of agent turns per iteration
- `--allowedTools <tools>` — restrict which tools the agent can use
