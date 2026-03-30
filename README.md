# lightctl

LightRun CLI -- runtime debugging from the terminal.

`lightctl` is a thin, token-efficient wrapper around the LightRun REST API. It
is designed for both interactive use and integration with AI coding agents that
need to instrument running JVMs.

> **Security warning:** Your LightRun API key grants the ability to instrument
> production JVMs -- treat it as an infrastructure credential. `lightctl` stores
> it in the OS keychain by default. Never commit it to version control.

## Installation

### Homebrew (macOS/Linux)

```bash
brew install aviadshiber/tap/lightctl
```

### Shell script

```bash
curl -fsSL https://raw.githubusercontent.com/aviadshiber/lightctl/main/install.sh | bash
```

### Debian/Ubuntu

Download the `.deb` from the [releases page](https://github.com/aviadshiber/lightctl/releases)
and install with `sudo dpkg -i lightctl_*.deb`.

### RPM (Fedora/RHEL)

Download the `.rpm` from the [releases page](https://github.com/aviadshiber/lightctl/releases)
and install with `sudo rpm -i lightctl_*.rpm`.

### Go install

```bash
go install github.com/aviadshiber/lightctl/cmd/lightctl@latest
```

### Build from source

```bash
git clone https://github.com/aviadshiber/lightctl.git
cd lightctl
make build
```

## Authentication

Set your API key (stored in the OS keychain):

```bash
lightctl config set api_key <your-key>
```

Or use an environment variable:

```bash
export LIGHTCTL_API_KEY=<your-key>
```

For CI environments without a keychain, use the plaintext fallback:

```bash
lightctl --insecure-plaintext-config config set api_key <your-key>
```

## Quick examples

```bash
# List agents
lightctl agents list
lightctl agents list --output table

# Add a snapshot
lightctl snapshot add <agent-id> com/example/Foo.java:42

# Add a conditional snapshot with expiry
lightctl snapshot add <agent-id> Foo.java:42 --condition "x > 10" --expire 300

# Watch for a variable value
lightctl watch <agent-id> Foo.java:42 "myVar"

# Clean up orphaned actions
lightctl gc
```

## Commands

### agents

```
lightctl agents list [--limit N] [--all]
```

List connected LightRun agents.

### snapshot

```
lightctl snapshot add <agent-id> <file>:<line> [--condition <expr>] [--expire <sec>] [--max-hits <n>]
lightctl snapshot list <agent-id> [--limit N] [--all]
lightctl snapshot get <agent-id> <snapshot-id>
lightctl snapshot delete <agent-id> <snapshot-id>
```

Manage snapshots (breakpoints that capture state without pausing).

### watch

```
lightctl watch <agent-id> <file>:<line> <expr> [--timeout <sec>] [--interval <sec>]
```

Create a temporary snapshot and poll until the given expression is captured.
Cleans up on success or signal. Exits with code 8 on timeout.

### gc

```
lightctl gc
```

Delete orphaned actions tracked in the local state file.

### config

```
lightctl config set <key> <value>
lightctl config get <key>
lightctl config list
```

Keys: `api_key`, `server`.

### version

```
lightctl version
```

## Output formatting

| Flag | Description |
|------|-------------|
| `--output json` | JSON output (default) |
| `--output table` | ASCII table output |
| `--pretty` | Pretty-print JSON |
| `--jq <expr>` | Filter JSON with a jq expression |
| `-q` / `--quiet` | Suppress informational messages |

## Environment variables

| Variable | Description | Default |
|----------|-------------|---------|
| `LIGHTCTL_API_KEY` | API key (overrides keychain/config) | -- |
| `LIGHTCTL_SERVER` | Server URL | `https://app.lightrun.com` |
| `LIGHTCTL_DEBUG` | Set to `1` to enable debug logging | `0` |
| `HTTP_PROXY` | HTTP proxy URL | -- |
| `HTTPS_PROXY` | HTTPS proxy URL | -- |
| `NO_COLOR` | Disable colored output | -- |
| `CLICOLOR` | Set to `0` to disable color | -- |

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Usage error (bad flags/arguments) |
| 8 | Watch timeout |
| 130 | Interrupted (SIGINT) |

## License

MIT
