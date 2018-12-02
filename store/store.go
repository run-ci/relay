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

// RelayStore is an interfacing defining the behavior of a fully-featured
// component for Relay.
type RelayStore interface {
	GitRepoStore
	PipelineStore
}

// GitRepoStore is anything that can hold data about source repositories.
type GitRepoStore interface {
	CreateGitRepo(GitRepo) error
	GetGitRepo(string, string) (GitRepo, error)
	GetGitRepos() ([]GitRepo, error)
}

// GitRepo is a Git repository.
type GitRepo struct {
	Remote string
	Branch string
}

// PipelineStore is an interface defining what a thing that can store
// pipelines should be able to do. All its members take pointers and
// update data in place instead of returning new values.
type PipelineStore interface {
	ReadPipeline(*Pipeline) error

	CreateRun(*Run) error
	CreateStep(*Step) error
	CreateTask(*Task) error

	UpdateRun(*Run) error
	UpdateStep(*Step) error
	UpdateTask(*Task) error
}

// Pipeline is a series of "runs" grouped together by a repository's URL
// and the pipeline's name.
type Pipeline struct {
	Remote string `db:"remote"`
	Name   string `db:"name"`
	Ref    string `db:"ref"`
	Runs   []Run  `db:"-"`
}

// Run is a representation of the actual state of execution of a pipeline.
type Run struct {
	Count   int        `db:"id"`
	Start   *time.Time `db:"start"`
	End     *time.Time `db:"end"`
	Success *bool      `db:"success"` // mid-run is neither success nor failure
	Steps   []Step     `db:"-"`

	PipelineRemote string `db:"pipeline_remote"`
	PipelineName   string `db:"pipeline_name"`
}

// Step is the representation of the actual state of execution of a group of
// pipeline tasks.
type Step struct {
	ID      int        `db:"id"`
	Name    string     `db:"name"`
	Start   *time.Time `db:"start"`
	End     *time.Time `db:"end"`
	Tasks   []Task     `db:"-"`
	Success *bool      `db:"success"` // mid-run is neither success nor failure

	PipelineRemote string `db:"pipeline_remote"`
	PipelineName   string `db:"pipeline_name"`
	RunCount       int    `db:"run_count"`
}

// Task is the representation of the actual state of execution of a pipeline
// run task.
type Task struct {
	ID      int        `db:"id"`
	Name    string     `db:"name"`
	Start   *time.Time `db:"start"`
	End     *time.Time `db:"end"`
	Success *bool      `db:"success"` // mid-run is neither success nor failure

	StepID int `db:"step_id"`
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
