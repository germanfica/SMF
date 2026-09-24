# Bash completion for the SMF CLI.
#
# This file has no dependency on bash-completion, so it can also be loaded
# directly from a checkout with: source completions/smf.bash

_smf_option_was_provided() {
  local OptionName="$1"
  local Word

  for Word in "${COMP_WORDS[@]}"; do
    case "$Word" in
      "$OptionName"|"$OptionName"=*)
        return 0
        ;;
    esac
  done
  return 1
}

_smf_complete_directories() {
  local CurrentWord="$1"
  local Candidate

  while IFS= read -r Candidate; do
    COMPREPLY+=("$Candidate")
  done < <(compgen -d -- "$CurrentWord")
}

_smf_complete_files() {
  local CurrentWord="$1"
  local Candidate

  while IFS= read -r Candidate; do
    COMPREPLY+=("$Candidate")
  done < <(compgen -f -- "$CurrentWord")
}

_smf_complete_ansible_playbook() {
  local CurrentWord="$1"
  local Candidate

  if [[ "$CurrentWord" == */* ]]; then
    _smf_complete_files "$CurrentWord"
    return
  fi

  while IFS= read -r Candidate; do
    COMPREPLY+=("$Candidate")
  done < <(compgen -c -- "$CurrentWord")
}

_smf_complete_install_options() {
  local CurrentWord="$1"
  local Candidate
  local -a AllOptions Candidates

  AllOptions=(
    --port
    --network-only
    --apply
    --disable-installer
    --project-dir
    --inventory
    --target
    --ask-become-pass
    --no-ask-become-pass
    --ask-vault-pass
    --no-ask-vault-pass
    --vault-password-file
    --ansible-playbook
    --ansible-runtime
    --no-bootstrap-ansible
    --non-interactive
    --help
  )

  Candidates=()
  for Candidate in "${AllOptions[@]}"; do
    if _smf_option_was_provided "$Candidate"; then
      continue
    fi

    case "$Candidate" in
      --network-only)
        _smf_option_was_provided --port && continue
        ;;
      --port)
        _smf_option_was_provided --network-only && continue
        ;;
      --ask-become-pass)
        _smf_option_was_provided --no-ask-become-pass && continue
        ;;
      --no-ask-become-pass)
        _smf_option_was_provided --ask-become-pass && continue
        ;;
      --ask-vault-pass)
        _smf_option_was_provided --no-ask-vault-pass && continue
        _smf_option_was_provided --vault-password-file && continue
        ;;
      --no-ask-vault-pass)
        _smf_option_was_provided --ask-vault-pass && continue
        ;;
      --vault-password-file)
        _smf_option_was_provided --ask-vault-pass && continue
        ;;
    esac

    Candidates+=("$Candidate")
  done

  COMPREPLY=( $(compgen -W "${Candidates[*]}" -- "$CurrentWord") )
}

_smf_complete() {
  local CurrentWord PreviousWord CommandName

  COMPREPLY=()
  CurrentWord="${COMP_WORDS[COMP_CWORD]}"
  PreviousWord="${COMP_WORDS[COMP_CWORD - 1]:-}"

  # Keep a single dash quiet, matching the usual Git completion behavior.
  [[ "$CurrentWord" == "-" ]] && return 0

  if (( COMP_CWORD == 1 )); then
    COMPREPLY=( $(compgen -W 'install list --help --version' -- "$CurrentWord") )
    return 0
  fi

  CommandName="${COMP_WORDS[1]}"
  case "$CommandName" in
    install)
      case "$PreviousWord" in
        --port|--target)
          return 0
          ;;
        --project-dir|--ansible-runtime)
          _smf_complete_directories "$CurrentWord"
          return 0
          ;;
        --inventory|--vault-password-file)
          _smf_complete_files "$CurrentWord"
          return 0
          ;;
        --ansible-playbook)
          _smf_complete_ansible_playbook "$CurrentWord"
          return 0
          ;;
      esac
      _smf_complete_install_options "$CurrentWord"
      ;;
    list)
      case "$PreviousWord" in
        --deployment-dir)
          _smf_complete_directories "$CurrentWord"
          return 0
          ;;
      esac
      COMPREPLY=( $(compgen -W '--deployment-dir --help' -- "$CurrentWord") )
      ;;
  esac
}

# Register both the installed command and the checkout invocation used during
# development. A sourced file can therefore complete `./smf install --` too.
complete -F _smf_complete smf ./smf
