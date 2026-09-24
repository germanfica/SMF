package main

import "testing"

func TestFormatCommandOperationQuotesWhitespaceAndQuotes(t *testing.T) {
	Operation := CommandOperation{
		ExecutablePath: "/usr/bin/example",
		Arguments:      []string{"plain", "two words", "it's"},
		WorkingDirectory: "/tmp/smf checkout",
	}
	FormattedOperation := FormatCommandOperation(Operation)
	ExpectedOperation := "(cd '/tmp/smf checkout' && /usr/bin/example plain 'two words' 'it'\"'\"'s')"
	if FormattedOperation != ExpectedOperation {
		t.Fatalf("formatted operation = %q, want %q", FormattedOperation, ExpectedOperation)
	}
}
