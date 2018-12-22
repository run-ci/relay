package http

import (
	"context"
	"net/http"

	"github.com/run-ci/relay/store"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

var logger *logrus.Entry

type ctxkey int

const (
	keyReqID ctxkey = iota
)

func init() {
	logger = logrus.WithField("package", "http")
}

// apiStore is a grouping of the minimum number of store
// interfaces the API needs to work.
type apiStore interface {
	GetPipelines(pid int) ([]store.Pipeline, error)
	GetPipeline(id int) (store.Pipeline, error)
	GetRun(pid, id int) (store.Run, error)
	GetStep(id int) (store.Step, error)
	GetTask(id int) (store.Task, error)

	CreateProject(*store.Project) error
	GetProject(id int) (store.Project, error)
	GetProjects() ([]store.Project, error)
}

// Server is a net/http.Server with dependencies like
// the database connection.
type Server struct {
	st     apiStore
	pollch chan<- []byte

	*http.Server
}

// NewServer returns a Server with a reference to `st`, listening
// on `addr`.
func NewServer(addr string, pollch chan<- []byte, st apiStore) *Server {
	srv := &Server{
		Server: &http.Server{
			Addr: addr,
		},

		st:     st,
		pollch: pollch,
	}

	r := mux.NewRouter()
	srv.Handler = r

	r.Handle("/", chain(getRoot, setRequestID, logRequest)).
		Methods(http.MethodGet)

	r.Handle("/projects", chain(srv.handleCreateProject, setRequestID, logRequest)).
		Methods(http.MethodPost)

	r.Handle("/projects", chain(srv.handleGetProjects, setRequestID, logRequest)).
		Methods(http.MethodGet)

	r.Handle("/projects/{id}", chain(srv.handleGetProject, setRequestID, logRequest)).
		Methods(http.MethodGet)

	// TODO: delete projects

	// TODO: create git remote for project

	r.Handle("/projects/{project_id}/pipelines", chain(srv.handleGetPipelines, setRequestID, logRequest)).
		Methods(http.MethodGet)

	r.Handle("/pipelines/{id}", chain(srv.handleGetPipeline, setRequestID, logRequest)).
		Methods(http.MethodGet)

	r.Handle("/pipelines/{pid}/runs/{count}", chain(srv.handleGetRun, setRequestID, logRequest)).
		Methods(http.MethodGet)

	r.Handle("/steps/{id}", chain(srv.handleGetStep, setRequestID, logRequest)).
		Methods(http.MethodGet)

	r.Handle("/tasks/{id}", chain(srv.handleGetTask, setRequestID, logRequest)).
		Methods(http.MethodGet)

	return srv
}

// Middleware is a function that can intercept the handling of an HTTP request
// to do something useful.
type middleware func(http.HandlerFunc) http.HandlerFunc

// Chain builds the final http.Handler from all the middlewares passed to it.
func chain(f http.HandlerFunc, mw ...middleware) http.Handler {
	// Because function calls are placed on a stack, they need to
	// be applied in reverse order from what they are passed in,
	// in order for calls to Chain() to be intuitive.
	for i := len(mw) - 1; i >= 0; i-- {
		f = mw[i](f)
	}

	return f
}

// SetRequestID sets a UUID on the request so that it can be tracked through
// logs, metrics and instrumentation.
func setRequestID(f http.HandlerFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		id := uuid.New().String()

		ctx := context.WithValue(req.Context(), keyReqID, id)
		logger.WithField("request_id", id).
			Debug("setting request ID")

		f(rw, req.WithContext(ctx))
	}
}

// LogRequest logs useful information about the request. It must have a
// "request_id" set on the request context.
func logRequest(f http.HandlerFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		reqid := req.Context().Value(keyReqID).(string)

		logger := logger.WithField("request_id", reqid)

		logger.Infof("%v %v", req.Method, req.URL)

		f(rw, req)
	}
}
