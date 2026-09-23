package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type CommandOperation struct {
	Name             string
	ExecutablePath   string
	Arguments        []string
	WorkingDirectory string
}

func ExecuteCommandOperation(Operation CommandOperation) error {
	Command := exec.Command(Operation.ExecutablePath, Operation.Arguments...)
	Command.Dir = Operation.WorkingDirectory
	Command.Stdin = os.Stdin
	Command.Stdout = os.Stdout
	Command.Stderr = os.Stderr
	if CommandError := Command.Run(); CommandError != nil {
		return fmt.Errorf("%s failed: %w", Operation.Name, CommandError)
	}
	return nil
}

func FormatCommandOperation(Operation CommandOperation) string {
	CommandParts := []string{QuoteShellArgument(Operation.ExecutablePath)}
	for _, Argument := range Operation.Arguments {
		CommandParts = append(CommandParts, QuoteShellArgument(Argument))
	}
	CommandLine := strings.Join(CommandParts, " ")
	if Operation.WorkingDirectory == "" {
		return CommandLine
	}
	return "(cd " + QuoteShellArgument(Operation.WorkingDirectory) + " && " + CommandLine + ")"
}

func QuoteShellArgument(Value string) string {
	if Value == "" {
		return "''"
	}
	for _, Character := range Value {
		if !(Character >= 'a' && Character <= 'z') &&
			!(Character >= 'A' && Character <= 'Z') &&
			!(Character >= '0' && Character <= '9') &&
			!strings.ContainsRune("%+,-./:=@_", Character) {
			return "'" + strings.ReplaceAll(Value, "'", "'\"'\"'") + "'"
		}
	}
	return Value
}
