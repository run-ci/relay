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
	Name  string `json:"name"`
	Tasks []Task `json:"tasks"`
}

// Task is a run task.
type Task struct {
	run.Task

	// Shadowing this because an argument in a normal run task
	// isn't the value of the actual argument, but a set of
	// sources where that value can be obtained. In a pipeline
	// run, it's necessary to have the actual value.
	Arguments map[string]interface{} `json:"arguments"`
}
