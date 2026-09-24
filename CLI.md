# SMF CLI and installation script

`install.sh` and the Go CLI have deliberately separate jobs.

| Component | Responsibility |
| --- | --- |
| `install.sh` | Build and install or update the `smf` binary from this checkout. |
| `smf` | Install, reconfigure, and inspect SMF through Ansible. |
| Ansible playbooks | Install Docker, build the pinned image, and deploy Compose resources. |

The shell script does not deploy a forum. This keeps its bootstrap logic
small and lets the Go CLI own validation, prompting, and future operational
commands.

## Install the CLI

Run the script from the root of the SMF checkout:

```bash
./install.sh
```

It builds the checked-out source with Go 1.17 or newer and installs `smf` to
`/usr/local/bin` by default. It uses `sudo` only for the final system-wide
file copies, so `smf --help` is available immediately on a standard Ubuntu
shell. It does not use `go install` and does not change Go's global package
state.

For a user-only installation that does not request `sudo`, use:

```bash
./install.sh --user
```

This installs the binary in `~/.local/bin`. If that directory is not already
in your `PATH`, add this line to `.bashrc` or `.zshrc`, then start a new shell:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

To select another writable directory without an elevation request, use:

```bash
./install.sh --bin-dir "$HOME/bin"
```

## Shell completion

The installation script also installs completion scripts for Bash and Zsh. They complete
the supported command or option for the current position: `smf --` offers only
global options, while `smf install --` and `smf configure --` offer only
options valid for their respective command. In particular, `--version` is
deliberately not suggested after an operation command.

The default system-wide installation places them in the standard
`/usr/local/share` completion directories, so shells with their usual
completion support enabled discover them without an additional `fpath` entry.
User and explicit-directory installations place them under `$XDG_DATA_HOME`
(or `~/.local/share`).

For an immediate Bash activation after a user or explicit-directory
installation, including completion of the checkout command `./smf`, run:

```bash
source "${BASH_COMPLETION_USER_DIR:-$HOME/.local/share/bash-completion}/completions/smf"
```

After the default system-wide installation, use its system completion path:

```bash
source /usr/local/share/bash-completion/completions/smf
```

If you are using the checkout before running `install.sh`, load its bundled
script directly instead:

```bash
source completions/smf.bash
```

For Zsh, add the installation directory to `fpath` and initialize completion
once in `.zshrc`:

```zsh
fpath=("${XDG_DATA_HOME:-$HOME/.local/share}/zsh/site-functions" $fpath)
autoload -Uz compinit
compinit
```

To activate the checkout script only for the current Zsh session, run:

```zsh
autoload -Uz compinit && compinit
source completions/_smf
```

Use `./install.sh --no-completions` when you do not want the script to
copy these scripts.

## Install SMF

Run the CLI from the SMF checkout, or pass `--project-dir` explicitly. It looks
for `playbooks/install-all.yml` from the working directory upward, so an
installed binary remains tied to the checkout that supplies its Docker files,
inventory, and Vault values.

```bash
smf install
```

In a terminal, this asks how SMF should be exposed and asks for confirmation.
The default is Docker-network-only exposure, which keeps the service on its
internal port `80` for a reverse proxy such as Nginx Proxy Manager.

For a scripted default installation:

```bash
smf install --non-interactive --apply
```

To publish a host port for direct access:

```bash
smf install --port 8080 --apply
```

This renders a managed Compose override containing:

```yaml
services:
  smf:
    ports:
      - "8080:80"
```

The base Compose file remains responsible for `expose: ["80"]`; there is no
flag to remove or alter that internal service port. The managed override
replaces any previously rendered `ports` list, so changing from `8080` to
`8090`, or returning to Docker-network-only mode, removes the old host
publication instead of retaining it. The deployment requires Docker Compose
2.24.4 or newer and verifies the resulting host-port state after it starts
SMF. A published port is bound on all host interfaces, so place it behind the
intended firewall or reverse proxy policy.

## Reconfigure an existing deployment

Use `configure` after the forum has been installed to change its Docker
exposure and its public URL. It does not run `install.php`, does not populate
the database, and does not create an administrator account again.

```bash
smf configure
```

In a terminal, the command asks how SMF should be exposed, asks for the forum
URL, prints the plan, and asks for confirmation. The forum URL must start with
`http://` or `https://` and must not have a trailing slash.

For direct access on a new port:

```bash
smf configure --port 8093 --forum-url http://localhost:8093
```

For a reverse proxy that reaches SMF through its Docker network only:

```bash
smf configure --network-only --forum-url https://forum.example.com
```

`--port` and `--network-only` are mutually exclusive. They are both explicit
in non-interactive mode, as is the public URL:

```bash
smf configure --port 8093 \
  --forum-url http://localhost:8093 \
  --non-interactive --apply
```

The command obtains the active SMF image from the resolved Compose
configuration, rewrites only the managed SMF override, recreates only the
`smf` service, and updates `$boardurl` in the persisted `Settings.php` target
in the `smf-config` volume. It also updates existing URL configuration records
for local avatars, custom avatars, smileys, and global theme and image paths.
Only values derived from the previous forum URL are rewritten; an external CDN
URL is preserved. If an earlier incomplete port-only reconfiguration already
changed `$boardurl`, the command can also repair standard local SMF asset paths
on the same scheme and host.

It verifies the selected host-port state and every updated URL value before
reporting success. It does not run `install.php`, create, drop, or populate
tables, recreate the administrator account, change posts or users, or recreate
the database service. It updates existing configuration records only.

## Ansible bootstrap

If `ansible-playbook` is already in `PATH`, the CLI uses it. Otherwise it
creates a private `ansible-core` runtime under
`$XDG_STATE_HOME/smf/ansible-core-2.21.4` (or
`~/.local/state/smf/ansible-core-2.21.4`). This keeps Ansible out of the system
Python environment. If the selected Python lacks `venv`, the applied command
installs only Ubuntu or Debian's `python3-venv` package with `sudo` before
creating that environment.

Use `--no-bootstrap-ansible` to require a preinstalled Ansible executable, or
`--ansible-playbook` to select one explicitly.

The CLI passes `SMF_DOCKER_COMMAND_BECOME=true` to the build playbook. This is
important on a clean machine: adding the user to Docker's group during the same
Ansible process does not change that process's supplementary groups. Only the
Docker commands in the image-build playbook use `become`; archive file work
continues as the invoking user.

## Vault, sudo, and initial setup

When `inventory/group_vars/smf/vault.yml` begins with an Ansible Vault header,
the CLI asks for its password automatically. You can instead use
`--vault-password-file`, `--ask-vault-pass`, or `--no-ask-vault-pass`.

The CLI also asks Ansible for the sudo password by default when it is not
already running as root. Override that with `--ask-become-pass` or
`--no-ask-become-pass`.

The initial command enables SMF's web installer. After completing the browser
setup, deploy again with the installer disabled. This creates a deployment
image derived from the pinned SMF image with `install.php` removed, then
verifies that the running service no longer contains that file. It also keeps
the existing Apache protection marker disabled. Keep the same `--port` option
when you use one:

```bash
smf install --disable-installer --port 8080 --apply
```

For Docker-network-only exposure, omit `--port`:

```bash
smf install --disable-installer --apply
```

## Inspect a local deployment

```bash
smf list
```

This runs `docker compose ... ps` against `/opt/smf`. Select a different local
deployment directory with `smf list --deployment-dir /path/to/smf`.
