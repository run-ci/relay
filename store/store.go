package store

import (
	"errors"
	"time"

	log "github.com/sirupsen/logrus"
)

var logger *log.Entry

// ErrPipelineNotFound is what's returned when a pipeline couldn't
// be found in the store.
var ErrPipelineNotFound = errors.New("pipeline not found")

func init() {
	logger = log.WithFields(log.Fields{
		"package": "store",
	})
}

// RelayStore is an all-encompassing interface for all the behaviors
// a store can exhibit.
type RelayStore interface {
	// CreateProject saves a project in the store, setting whatever
	// values on the input that need to be set at create-time.
	CreateProject(*Project) error
	// GetProject returns a Project with its GitRemotes. It doesn't
	// fetch the actual pipelines in those remotes.
	GetProject(id int) (Project, error)
	// GetProjects returns a preview list of Projects, without any
	// information as to what's inside those Projects.
	GetProjects() ([]Project, error)

	GetPipelines(filter GitRemote) ([]Pipeline, error)
	GetPipeline(id int) (Pipeline, error)

	// These Create* methods save their respective resources in
	// the store, setting create-time values on the input.
	CreateRun(*Run) error
	CreateStep(*Step) error
	CreateTask(*Task) error

	// These Update* methods update their respective resources in
	// the store, setting update-time values on the input if there
	// are any.
	UpdateRun(*Run) error
	UpdateStep(*Step) error
	UpdateTask(*Task) error
}

// Project is a grouping of different pipelines by their remotes.
type Project struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`

	GitRemotes []GitRemote `json:"git_remotes"`
}

// GitRemote is the remote location of a Git repository, specified
// by the URL and branch name.
type GitRemote struct {
	URL    string `json:"url"`
	Branch string `json:"branch"`

	Pipelines []Pipeline `json:"pipelines,omitempty"`
}

// Pipeline is a grouping of steps with a name associated.
type Pipeline struct {
	ID   int    `json:"id"`
	Name string `json:"name"`

	GitRemote GitRemote `json:"git_remote"`

	// The steps are accessed run by run because a pipeline
	// can be updated to have different steps. Placing them
	// directly on the pipeline itself would mean that the
	// data from previous runs could be mangled.
	Runs []Run `json:"runs"`
}

// Run is a representation of the actual state of execution of a pipeline.
type Run struct {
	Count   int        `json:"count"`
	Start   *time.Time `json:"start"`
	End     *time.Time `json:"end"`
	Success *bool      `json:"success"` // mid-run is neither success nor failure

	// This attribute is necessary to have here because a run can only be
	// identified by the combination of its pipeline and its place.
	PipelineID int `json:"pipeline_id"`

	Steps []Step `json:"steps"`
}

// Step is the representation of the actual state of execution of a group of
// pipeline tasks.
type Step struct {
	ID      int        `json:"id"`
	Name    string     `json:"name"`
	Start   *time.Time `json:"start"`
	End     *time.Time `json:"end"`
	Tasks   []Task     `json:"tasks"`
	Success *bool      `json:"success"` // mid-run is neither success nor failure

	PipelineID int `json:"-"`
	RunCount   int `json:"-"`
}

// Task is the representation of the actual state of execution of a pipeline
// run task.
type Task struct {
	ID      int        `json:"id"`
	Name    string     `json:"name"`
	Start   *time.Time `json:"start"`
	End     *time.Time `json:"end"`
	Success *bool      `json:"success"` // mid-run is neither success nor failure

	StepID int `json:"-"`
}

// SetStart is a convenience method for setting the start time pointer.
func (r *Run) SetStart() {
	t := time.Now()
	r.Start = &t
}

// SetEnd is a convenience method for setting the end time pointer.
func (r *Run) SetEnd() {
	t := time.Now()
	r.End = &t
}

// MarkSuccess is a convenience method for setting the success status.
func (r *Run) MarkSuccess(s bool) {
	r.Success = &s
}

// Failed is a convenience method for checking the success status
// for a failure.
func (r *Run) Failed() bool {
	return r.Success != nil && *r.Success == false
}

// SetStart is a convenience method for setting the start time pointer.
func (st *Step) SetStart() {
	t := time.Now()
	st.Start = &t
}

// SetEnd is a convenience method for setting the end time pointer.
func (st *Step) SetEnd() {
	t := time.Now()
	st.End = &t
}

// MarkSuccess is a convenience method for setting the success status.
func (st *Step) MarkSuccess(s bool) {
	st.Success = &s
}

// SetStart is a convenience method for setting the start time pointer.
func (task *Task) SetStart() {
	t := time.Now()
	task.Start = &t
}

// SetEnd is a convenience method for setting the end time pointer.
func (task *Task) SetEnd() {
	t := time.Now()
	task.End = &t
}

// MarkSuccess is a convenience method for setting the success status.
func (task *Task) MarkSuccess(s bool) {
	task.Success = &s
}
