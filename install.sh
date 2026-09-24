#!/bin/sh
# Build the checked-out SMF CLI and install it without changing Go's global
# package state. This script intentionally does not deploy SMF itself.
set -eu

ProgramName="smf"
SourceDirectoryPath=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
InstallationDirectoryPath="/usr/local/bin"
InstallationScope="system"
DataDirectoryPath="${XDG_DATA_HOME:-$HOME/.local/share}"
InstallCompletions=true

PrintUsage() {
  cat <<'EOF'
Usage: ./install.sh [--user] [--bin-dir PATH] [--no-completions]

Builds the smf CLI from this checkout and installs or updates it.

Options:
  --user              Install in ~/.local/bin without sudo.
  --bin-dir PATH      Install in PATH using the current user's permissions.
  --no-completions    Do not install Bash and Zsh completion scripts.
  --help              Show this help.

Without an option, the script installs in /usr/local/bin and uses sudo only
for the final system-wide file copies. It needs Go 1.17 or newer and does not
run the SMF deployment.
EOF
}

PrintError() {
  printf 'install.sh: %s\n' "$1" >&2
}

RunPrivileged() {
  if [ "$(id -u)" -eq 0 ]; then
    "$@"
    return
  fi
  if ! command -v sudo >/dev/null 2>&1; then
    PrintError 'sudo is required for the default system-wide installation.'
    exit 1
  fi
  sudo "$@"
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --user)
      if [ "$InstallationScope" = custom ]; then
        PrintError '--user and --bin-dir cannot be used together'
        exit 2
      fi
      InstallationDirectoryPath="$HOME/.local/bin"
      InstallationScope="user"
      shift
      ;;
    --bin-dir)
      if [ "$#" -lt 2 ]; then
        PrintError '--bin-dir requires a path'
        exit 2
      fi
      if [ -z "$2" ]; then
        PrintError '--bin-dir requires a non-empty path'
        exit 2
      fi
      if [ "$InstallationScope" = user ]; then
        PrintError '--user and --bin-dir cannot be used together'
        exit 2
      fi
      InstallationDirectoryPath="$2"
      InstallationScope="custom"
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
      PrintError "unknown option: $1"
      PrintUsage >&2
      exit 2
      ;;
  esac
done

if ! command -v go >/dev/null 2>&1; then
  PrintError 'Go 1.17 or newer is required to build smf.'
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

if [ "$InstallationScope" = system ]; then
  BashCompletionPath="/usr/local/share/bash-completion/completions/$ProgramName"
  ZshCompletionPath="/usr/local/share/zsh/site-functions/_$ProgramName"
else
  BashCompletionRootPath="${BASH_COMPLETION_USER_DIR:-$DataDirectoryPath/bash-completion}"
  BashCompletionPath="$BashCompletionRootPath/completions/$ProgramName"
  ZshCompletionPath="$DataDirectoryPath/zsh/site-functions/_$ProgramName"
fi

BashCompletionSourcePath="$SourceDirectoryPath/completions/$ProgramName.bash"
ZshCompletionSourcePath="$SourceDirectoryPath/completions/_$ProgramName"
if [ "$InstallCompletions" = true ] && { [ ! -r "$BashCompletionSourcePath" ] || [ ! -r "$ZshCompletionSourcePath" ]; }; then
  PrintError 'bundled shell completion scripts are missing.'
  exit 1
fi

if [ "$InstallationScope" = system ]; then
  RunPrivileged mkdir -p "$InstallationDirectoryPath"
  RunPrivileged install -m 0755 "$TemporaryDirectoryPath/$ProgramName" "$InstallationDirectoryPath/$ProgramName"
  if [ "$InstallCompletions" = true ]; then
    RunPrivileged mkdir -p "$(dirname -- "$BashCompletionPath")" "$(dirname -- "$ZshCompletionPath")"
    RunPrivileged install -m 0644 "$BashCompletionSourcePath" "$BashCompletionPath"
    RunPrivileged install -m 0644 "$ZshCompletionSourcePath" "$ZshCompletionPath"
  fi
else
  mkdir -p "$InstallationDirectoryPath"
  install -m 0755 "$TemporaryDirectoryPath/$ProgramName" "$InstallationDirectoryPath/$ProgramName"
  if [ "$InstallCompletions" = true ]; then
    mkdir -p "$(dirname -- "$BashCompletionPath")" "$(dirname -- "$ZshCompletionPath")"
    install -m 0644 "$BashCompletionSourcePath" "$BashCompletionPath"
    install -m 0644 "$ZshCompletionSourcePath" "$ZshCompletionPath"
  fi
fi

printf 'Installed %s (%s) in %s\n' "$ProgramName" "$BuildVersion" "$InstallationDirectoryPath"
if [ "$InstallCompletions" = true ]; then
  printf 'Installed Bash completion in %s\n' "$BashCompletionPath"
  printf 'Installed Zsh completion in %s\n' "$ZshCompletionPath"
  printf '%s\n' 'Open a new Bash or Zsh session to load shell completion.'
fi
case ":${PATH}:" in
  *":${InstallationDirectoryPath}:"*) ;;
  *)
    printf 'Add %s to PATH before invoking %s from another directory.\n' "$InstallationDirectoryPath" "$ProgramName"
    ;;
esac
