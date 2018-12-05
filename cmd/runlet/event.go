package main

import (
	"github.com/run-ci/relay/store"
	"github.com/run-ci/run/pkg/run"
)

// Event is a message that comes in requesting a pipeline run.
type Event struct {
	GitRemote store.GitRemote `json:"git_remote"`
	Name      string          `json:"name"`
	Steps     []Step          `json:"steps"`
}

// Step is a grouping of tasks that can be run in parallel.
type Step struct {
	Name  string     `json:"name"`
	Tasks []run.Task `json:"tasks"`
}
