#!/bin/sh
# Remove an SMF CLI installation created by install.sh. This script does not
# remove an SMF deployment, Docker images, volumes, or configuration.
set -eu

ProgramName="smf"
InstallationDirectoryPath="/usr/local/bin"
InstallationScope="system"
DataDirectoryPath="${XDG_DATA_HOME:-$HOME/.local/share}"

PrintUsage() {
  cat <<'EOF'
Usage: ./uninstall.sh [--user] [--bin-dir PATH]

Removes the smf CLI binary and its Bash and Zsh completion scripts.

Options:
  --user              Remove from ~/.local/bin without sudo.
  --bin-dir PATH      Remove from PATH using the current user's permissions.
  --help              Show this help.

Without an option, the script removes the system-wide installation from
/usr/local/bin and uses sudo only for the final file removals. It does not
remove an SMF deployment, Docker images, volumes, or configuration.
EOF
}

PrintError() {
  printf 'uninstall.sh: %s\n' "$1" >&2
}

RunPrivileged() {
  if [ "$(id -u)" -eq 0 ]; then
    "$@"
    return
  fi
  if ! command -v sudo >/dev/null 2>&1; then
    PrintError 'sudo is required for the default system-wide uninstallation.'
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
      if [ "$#" -lt 2 ] || [ -z "$2" ]; then
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

if [ "$InstallationScope" = system ]; then
  BashCompletionPath="/usr/local/share/bash-completion/completions/$ProgramName"
  ZshCompletionPath="/usr/local/share/zsh/site-functions/_$ProgramName"
else
  BashCompletionRootPath="${BASH_COMPLETION_USER_DIR:-$DataDirectoryPath/bash-completion}"
  BashCompletionPath="$BashCompletionRootPath/completions/$ProgramName"
  ZshCompletionPath="$DataDirectoryPath/zsh/site-functions/_$ProgramName"
fi

BinaryPath="$InstallationDirectoryPath/$ProgramName"
if [ "$InstallationScope" = system ]; then
  RunPrivileged rm -f -- "$BinaryPath" "$BashCompletionPath" "$ZshCompletionPath"
else
  rm -f -- "$BinaryPath" "$BashCompletionPath" "$ZshCompletionPath"
fi

printf 'Removed %s from %s\n' "$ProgramName" "$InstallationDirectoryPath"
printf 'Removed Bash completion from %s\n' "$BashCompletionPath"
printf 'Removed Zsh completion from %s\n' "$ZshCompletionPath"
