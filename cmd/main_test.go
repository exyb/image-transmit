package main

import (
	"testing"
)

func TestFindStrInSlice(t *testing.T) {
	slice := []string{"a", "b", "c"}

	idx, found := findStrInSlice(slice, "b")
	if !found || idx != 1 {
		t.Errorf("findStrInSlice(..., \"b\") = (%d, %v), want (1, true)", idx, found)
	}

	idx, found = findStrInSlice(slice, "d")
	if found || idx != -1 {
		t.Errorf("findStrInSlice(..., \"d\") = (%d, %v), want (-1, false)", idx, found)
	}
}

func TestAtoi(t *testing.T) {
	if got := atoi("42"); got != 42 {
		t.Errorf("atoi(\"42\") = %d, want 42", got)
	}
	if got := atoi("invalid"); got != 0 {
		t.Errorf("atoi(\"invalid\") = %d, want 0", got)
	}
	if got := atoi(""); got != 0 {
		t.Errorf("atoi(\"\") = %d, want 0", got)
	}
}

func TestGetInputList(t *testing.T) {
	var result []string
	getInputList("image1:latest\nimage2:v1.0\n\nimage3", &result)

	if len(result) != 3 {
		t.Fatalf("Expected 3 images, got %d", len(result))
	}
	if result[0] != "image1:latest" {
		t.Errorf("result[0] = %q, want image1:latest", result[0])
	}
	if result[1] != "image2:v1.0" {
		t.Errorf("result[1] = %q, want image2:v1.0", result[1])
	}
	if result[2] != "image3" {
		t.Errorf("result[2] = %q, want image3", result[2])
	}
}
