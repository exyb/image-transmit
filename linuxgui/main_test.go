//go:build linux

package main

import (
	"testing"

	. "github.com/wct-devops/image-transmit/core"
)

func TestRepoDisplayName(t *testing.T) {
	if got := repoDisplayName(Repo{Name: "test"}); got != "test" {
		t.Errorf("repoDisplayName({Name:test}) = %q, want test", got)
	}
	if got := repoDisplayName(Repo{Registry: "hub.docker.com", Repository: "library"}); got != "hub.docker.com-library" {
		t.Errorf("repoDisplayName = %q, want hub.docker.com-library", got)
	}
	if got := repoDisplayName(Repo{Registry: "hub.docker.com"}); got != "hub.docker.com" {
		t.Errorf("repoDisplayName = %q, want hub.docker.com", got)
	}
}

func TestFindRepoByName(t *testing.T) {
	repos := []Repo{
		{Name: "a", Registry: "reg1"},
		{Name: "b", Registry: "reg2"},
	}

	if got := findRepoByName(repos, "a"); got == nil || got.Name != "a" {
		t.Errorf("findRepoByName(..., \"a\") = %v, want repo a", got)
	}
	if got := findRepoByName(repos, "c"); got != nil {
		t.Errorf("findRepoByName(..., \"c\") = %v, want nil", got)
	}
}
