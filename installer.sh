#!/bin/sh
# Build the checked-out SMF CLI and install it without changing Go's global
# package state. This installer intentionally does not deploy SMF itself.
set -eu

ProgramName="smf"
SourceDirectoryPath=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
InstallationDirectoryPath="${SMF_INSTALL_DIR:-$HOME/.local/bin}"
DataDirectoryPath="${XDG_DATA_HOME:-$HOME/.local/share}"
BashCompletionRootPath="${BASH_COMPLETION_USER_DIR:-$DataDirectoryPath/bash-completion}"
BashCompletionPath="$BashCompletionRootPath/completions/$ProgramName"
ZshCompletionPath="$DataDirectoryPath/zsh/site-functions/_$ProgramName"
InstallCompletions=true

PrintUsage() {
  cat <<'EOF'
Usage: ./installer.sh [--bin-dir PATH] [--no-completions]

Builds the smf CLI from this checkout and installs or updates it.

Options:
  --bin-dir PATH      Install the binary in PATH instead of ~/.local/bin.
  --no-completions    Do not install Bash and Zsh completion scripts.
  --help              Show this help.

The script needs Go 1.17 or newer. It does not run the SMF deployment.
EOF
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --bin-dir)
      if [ "$#" -lt 2 ]; then
        printf '%s\n' 'installer.sh: --bin-dir requires a path' >&2
        exit 2
      fi
      InstallationDirectoryPath="$2"
      shift 2
      ;;
    --no-completions)
      InstallCompletions=false
      shift
      ;;
    --help|-h)
      PrintUsage
      exit 0
      ;;
    *)
      printf 'installer.sh: unknown option: %s\n' "$1" >&2
      PrintUsage >&2
      exit 2
      ;;
  esac
done

if ! command -v go >/dev/null 2>&1; then
  printf '%s\n' 'installer.sh: Go 1.17 or newer is required to build smf.' >&2
  exit 1
fi

BuildVersion="development"
if command -v git >/dev/null 2>&1 && git -C "$SourceDirectoryPath" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  BuildVersion=$(git -C "$SourceDirectoryPath" describe --tags --always --dirty 2>/dev/null || printf '%s' 'development')
fi

TemporaryDirectoryPath=$(mktemp -d "${TMPDIR:-/tmp}/smf-installer.XXXXXX")
trap 'rm -rf "$TemporaryDirectoryPath"' EXIT HUP INT TERM

go build \
  -trimpath \
  -ldflags "-X main.Version=$BuildVersion" \
  -o "$TemporaryDirectoryPath/$ProgramName" \
  "$SourceDirectoryPath"

mkdir -p "$InstallationDirectoryPath"
install -m 0755 "$TemporaryDirectoryPath/$ProgramName" "$InstallationDirectoryPath/$ProgramName"

if [ "$InstallCompletions" = true ]; then
  BashCompletionSourcePath="$SourceDirectoryPath/completions/$ProgramName.bash"
  ZshCompletionSourcePath="$SourceDirectoryPath/completions/_$ProgramName"
  if [ ! -r "$BashCompletionSourcePath" ] || [ ! -r "$ZshCompletionSourcePath" ]; then
    printf '%s\n' 'installer.sh: bundled shell completion scripts are missing.' >&2
    exit 1
  fi

  mkdir -p "$(dirname -- "$BashCompletionPath")" "$(dirname -- "$ZshCompletionPath")"
  install -m 0644 "$BashCompletionSourcePath" "$BashCompletionPath"
  install -m 0644 "$ZshCompletionSourcePath" "$ZshCompletionPath"
fi

printf 'Installed %s (%s) in %s\n' "$ProgramName" "$BuildVersion" "$InstallationDirectoryPath"
if [ "$InstallCompletions" = true ]; then
  printf 'Installed Bash completion in %s\n' "$BashCompletionPath"
  printf 'Installed Zsh completion in %s\n' "$ZshCompletionPath"
  printf '%s\n' 'Start a new Bash shell, or follow the Bash and Zsh activation commands in CLI.md.'
fi
case ":${PATH}:" in
  *":${InstallationDirectoryPath}:"*) ;;
  *)
    printf 'Add %s to PATH before invoking %s from another directory.\n' "$InstallationDirectoryPath" "$ProgramName"
    ;;
esac
