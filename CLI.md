# SMF CLI and installer

`installer.sh` and the Go CLI have deliberately separate jobs.

| Component | Responsibility |
| --- | --- |
| `installer.sh` | Build and install or update the `smf` binary from this checkout. |
| `smf` | Validate and run an SMF installation through Ansible. |
| Ansible playbooks | Install Docker, build the pinned image, and deploy Compose resources. |

The shell installer does not deploy a forum. This keeps its bootstrap logic
small and lets the Go CLI own validation, prompting, and future operational
commands.

## Install the CLI

Run the script from the root of the SMF checkout:

```bash
./installer.sh
```

It builds the checked-out source with Go 1.17 or newer and installs `smf` to
`~/.local/bin` by default. It does not use `go install` and does not change
Go's global package state.

Use another directory only when it is already appropriate for your PATH:

```bash
./installer.sh --bin-dir /usr/local/bin
```

The script does not elevate privileges itself. Use a writable directory or run
the command through the privilege mechanism you selected.

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

`--install` is a compatibility spelling for the same interactive flow:

```bash
smf --install
```

It shows the plan and asks for confirmation just like `smf install`. For a
scripted Docker-network-only deployment, add `--apply` and
`--non-interactive`:

```bash
smf --install --apply --non-interactive
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
