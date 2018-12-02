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

// PipelineQuerier is an interface defining the behavior of an entity
// that can query a store for pipeline information.
type PipelineQuerier interface {
	GetPipelines() ([]Pipeline, error)
	ReadPipeline(*Pipeline) error
}

// PipelineStore is an interface defining what a thing that can store
// pipelines should be able to do. All its members take pointers and
// update data in place instead of returning new values.
type PipelineStore interface {
	CreateRun(*Run) error
	CreateStep(*Step) error
	CreateTask(*Task) error

	UpdateRun(*Run) error
	UpdateStep(*Step) error
	UpdateTask(*Task) error

	PipelineQuerier
}

// Pipeline is a series of "runs" grouped together by a repository's URL
// and the pipeline's name.
type Pipeline struct {
	Remote string `json:"remote"`
	Name   string `json:"name"`
	Ref    string `json:"ref"`
	Runs   []Run  `json:"runs"`
}

// Run is a representation of the actual state of execution of a pipeline.
type Run struct {
	Count   int        `json:"count"`
	Start   *time.Time `json:"start"`
	End     *time.Time `json:"end"`
	Success *bool      `json:"success"` // mid-run is neither success nor failure
	Steps   []Step     `json:"steps"`

	PipelineRemote string `json:"-"`
	PipelineName   string `json:"-"`
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

	PipelineRemote string `json:"-"`
	PipelineName   string `json:"-"`
	RunCount       int    `json:"-"`
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
