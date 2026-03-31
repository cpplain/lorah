# Guide: Configuration

A `.lorah` directory contains the files that define a unit of work for the agent loop. It sits at the project root by default (overridable with `--dir`). Commit `.lorah/` to git — the workflow depends on git history for state and continuity, and task files are committed as part of the loop.

```
.lorah/
├── plan.md              # scope and acceptance criteria
├── prompt.md            # orient + route to phase prompt
├── prompts/
│   ├── plan.md          # task selection
│   ├── test.md          # write tests for selected task
│   └── implement.md     # make tests pass
├── tasks/
│   ├── 01-<task>.md     # one file per task
│   └── ...
└── settings.json        # Claude Code CLI settings
```

## Plan file

The plan file is the output of the scoping step — [Phase 1](workflow.md#phase-1-scope-the-work) in the spec-driven workflow. It defines what is being built and what done looks like. The agent loop uses it as the contract between the human and the agents.

A plan file contains:

- **Scope** — what is being built, at the level of a brief description and a list of capabilities. Reference the design specs rather than duplicating them.
- **Boundaries** — constraints and invariants that apply across the work (e.g., "stdlib only", "no external dependencies").
- **Acceptance criteria** — concrete, verifiable conditions that define when the work is complete. An agent should be able to check each criterion against git state and test results.

A plan file does not contain individual tasks. Task selection happens inside the loop, where each agent picks the next task based on current state.

```markdown
# <Project/Feature Name>

## Scope

What is being built — brief description and list of capabilities.
Reference the design specs rather than duplicating them.

## Boundaries

- Constraints and invariants that apply across the work.

## Acceptance Criteria

- [ ] Concrete, verifiable conditions.
```

## Prompt files

The prompt structure splits into a router prompt and phase-specific prompts. This keeps each agent's context small and focused. See the [prompt files guide](prompts.md) for templates.

## Task files

Each task gets its own file in `.lorah/tasks/`. The planning agent creates one task file per iteration; the testing and implementation agents update it as they work. See the [task files guide](tasks.md) for the template and status values.

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
  "attribution": { "commit": "", "pr": "" }
}
```

- **model** — which Claude model to use.
- **permissions** — `bypassPermissions` is typical for autonomous loops where no human is approving each action.
- **sandbox** — enables sandboxed execution. `autoAllowBashIfSandboxed` avoids permission prompts for shell commands when sandboxing is on.
- **attribution** — text added to commit messages (as git trailers) and PR descriptions. Empty strings disable attribution; omitting the field uses Claude Code's defaults.

See the [Claude Code settings reference](https://code.claude.com/docs/en/settings) for all available settings.

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

See the [Claude Code CLI reference](https://code.claude.com/docs/en/cli-reference) for all available flags.
