package utils

import (
	"testing"
)

func testStrVal(t *testing.T) {
	// Arrange
	str := "Some text for flavour"
	pointer := StringPtr("Some text for flavour")
	other := 1

	// Act
	convertedP := StrVal(pointer)
	convertedO := StrVal(other)

	if str != convertedP {
		t.Fatalf("Incorrect conversion of pointer")
	}
	if convertedO != "" {
		t.Fatalf("Incorrect conversion of type other than pointer")
	}
}
