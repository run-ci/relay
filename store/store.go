package store

import (
	"errors"
	"time"

	log "github.com/sirupsen/logrus"
)

var logger *log.Entry

var (
	// ErrPipelineNotFound is what's returned when a pipeline couldn't
	// be found in the store.
	ErrPipelineNotFound = errors.New("pipeline not found")
	// ErrNoPipelines is an error returned when a method of a RelayStore
	// doesn't find any pipelines.
	ErrNoPipelines = errors.New("no pipelines found")
	// ErrRunNotFound is an error returned when a run isn't found for a
	// given pipeline.
	ErrRunNotFound = errors.New("run not found")
	// ErrStepNotFound is an error returned when a Step isn't found.
	ErrStepNotFound = errors.New("step not found")
	// ErrTaskNotFound is an error returned when a Task isn't found.
	ErrTaskNotFound = errors.New("task not found")
)

func init() {
	logger = log.WithFields(log.Fields{
		"package": "store",
	})
}

// RelayStore is an all-encompassing interface for all the behaviors
// a store can exhibit. The interface is massive, but all this is included
// so that store implementations can be seamlessly swapped out. Consumers
// should define their own interfaces that use a subset of this interface's
// functions related to what they're interested in.
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

	GetPipelines(projectid int) ([]Pipeline, error)
	GetPipeline(id int) (Pipeline, error)
	// GetPipelineID takes these fields because it's the only way to
	// identify a pipeline before the ID is known. If there are no
	// pipelines matching these filters, implementations should return
	// ErrNoPipelines.
	GetPipelineID(GitRemote, string) (int, error)

	// GetRun returns the nth run for the pipeline with the passed
	// in ID from the store. If a run with that count isn't found
	// for whatever reason, ErrRunNotFound is returned.
	GetRun(pid, n int) (Run, error)
	// GetStep returns the step with the given ID from the store.
	// If no step with that ID is found, ErrStepNotFound should
	// be returned.
	GetStep(id int) (Step, error)
	// GetTask returns the Task with the given ID from the store.
	// If no Task with that ID is found, ErrTaskNotFound should
	// be returned.
	GetTask(id int) (Task, error)

	// These Create* methods save their respective resources in
	// the store, setting create-time values on the input.
	CreatePipeline(*Pipeline) error
	CreateRun(*Run) error
	CreateStep(*Step) error
	CreateTask(*Task) error

	// These Update* methods update their respective resources in
	// the store, setting update-time values on the input if there
	// are any.
	UpdatePipeline(*Pipeline) error
	UpdateRun(*Run) error
	UpdateStep(*Step) error
	UpdateTask(*Task) error

	CreateGroup(*Group) error
}

// Project is a grouping of different pipelines by their remotes.
type Project struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`

	GitRemotes []GitRemote `json:"git_remotes,omitempty"`
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
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Success *bool  `json:"success"`

	GitRemote GitRemote `json:"git_remote"`
	ProjectID int       `json:"project_id"`

	// The steps are accessed run by run because a pipeline
	// can be updated to have different steps. Placing them
	// directly on the pipeline itself would mean that the
	// data from previous runs could be mangled.
	Runs []Run `json:"runs,omitempty"`
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

	Steps []Step `json:"steps,omitempty"`
}

// Step is the representation of the actual state of execution of a group of
// pipeline tasks.
type Step struct {
	ID      int        `json:"id"`
	Name    string     `json:"name"`
	Start   *time.Time `json:"start"`
	End     *time.Time `json:"end"`
	Success *bool      `json:"success"` // mid-run is neither success nor failure

	PipelineID int `json:"-"`
	RunCount   int `json:"-"`

	Tasks []Task `json:"tasks,omitempty"`
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

// User is an entity that's authorized to interact with the CI system.
type User struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`

	Group Group `json:"group"`
}

// Group is an aggregate of users to make things like assigning permissions
// to multiple users easer.
type Group struct {
	Name string
}

// MarkSuccess is a convenience method for setting the success status.
func (p *Pipeline) MarkSuccess(s bool) {
	p.Success = &s
}

// Failed is a convenience method for checking the success status
// for a failure.
func (p *Pipeline) Failed() bool {
	return p.Success != nil && *p.Success == false
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
