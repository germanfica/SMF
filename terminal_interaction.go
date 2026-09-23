package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func StandardInputIsTerminal() bool {
	FileInformation, FileInformationError := os.Stdin.Stat()
	if FileInformationError != nil {
		return false
	}
	return FileInformation.Mode()&os.ModeCharDevice != 0
}

func PromptForExposure(Configuration CLIConfiguration, Reader *bufio.Reader) (CLIConfiguration, error) {
	fmt.Fprintln(os.Stdout, "How should SMF be exposed?")
	fmt.Fprintln(os.Stdout, "")
	fmt.Fprintln(os.Stdout, "1. Docker network only (recommended for a reverse proxy)")
	fmt.Fprintln(os.Stdout, "2. Publish a host port for direct access")
	fmt.Fprint(os.Stdout, "\nSelection [1]: ")
	Selection, SelectionError := ReadTerminalLine(Reader)
	if SelectionError != nil {
		return CLIConfiguration{}, SelectionError
	}
	switch Selection {
	case "", "1":
		Configuration.ExposureMode = ExposureModeNetworkOnly
		Configuration.PublishedPort = 0
	case "2":
		fmt.Fprint(os.Stdout, "Host port [8080]: ")
		PortText, PortTextError := ReadTerminalLine(Reader)
		if PortTextError != nil {
			return CLIConfiguration{}, PortTextError
		}
		if PortText == "" {
			PortText = "8080"
		}
		Port, PortError := strconv.Atoi(PortText)
		if PortError != nil || Port < 1 || Port > 65535 {
			return CLIConfiguration{}, fmt.Errorf("host port must be a number between 1 and 65535")
		}
		Configuration.ExposureMode = ExposureModePublishedPort
		Configuration.PublishedPort = Port
	default:
		return CLIConfiguration{}, fmt.Errorf("select 1 or 2")
	}
	Configuration.ExposureWasSpecified = true
	return Configuration, nil
}

func PromptForInstallationConfirmation(Reader *bufio.Reader) (bool, error) {
	fmt.Fprint(os.Stdout, "\nApply this plan now? [y/N]: ")
	Confirmation, ConfirmationError := ReadTerminalLine(Reader)
	if ConfirmationError != nil {
		return false, ConfirmationError
	}
	switch strings.ToLower(Confirmation) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

func ReadTerminalLine(Reader *bufio.Reader) (string, error) {
	Line, LineError := Reader.ReadString('\n')
	if LineError != nil && len(Line) == 0 {
		return "", fmt.Errorf("read terminal input: %w", LineError)
	}
	return strings.TrimSpace(Line), nil
}
