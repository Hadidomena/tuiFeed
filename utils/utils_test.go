package utils

import (
	"testing"
)

func TestStringPtr(t *testing.T) {
	// Arrange
	str := "some String"

	// Act
	res := StringPtr(str)

	// Assert
	if str != *res {
		t.Fatalf("Incorrect string returned")
	}
}

func TestStrVal(t *testing.T) {
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
