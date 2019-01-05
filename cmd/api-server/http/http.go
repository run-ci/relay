package http

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/run-ci/relay/store"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

var logger *logrus.Entry

type ctxkey int

const (
	keyReqID ctxkey = iota
	keyReqSub
)

func init() {
	logger = logrus.WithField("package", "http")
}

// apiStore is a grouping of the minimum number of store
// interfaces the API needs to work.
type apiStore interface {
	GetPipelines(user string, pid int) ([]store.Pipeline, error)
	GetPipeline(user string, id int) (store.Pipeline, error)
	GetRun(user string, pid, id int) (store.Run, error)
	GetStep(user string, id int) (store.Step, error)
	GetTask(user string, id int) (store.Task, error)

	CreateProject(*store.Project) error
	GetProject(user string, id int) (store.Project, error)
	GetProjects(user string) ([]store.Project, error)

	CreateGitRemote(user string, remote *store.GitRemote) error

	Authenticate(user, pass string) error
}

// Server is a net/http.Server with dependencies like
// the database connection.
type Server struct {
	st        apiStore
	pollch    chan<- []byte
	jwtsecret []byte

	*http.Server
}

// NewServer returns a Server with a reference to `st`, listening
// on `addr`.
func NewServer(addr string, pollch chan<- []byte, st apiStore, jwtsecret string) *Server {
	srv := &Server{
		Server: &http.Server{
			Addr: addr,
		},

		st:        st,
		pollch:    pollch,
		jwtsecret: []byte(jwtsecret),
	}

	r := mux.NewRouter()
	srv.Handler = r

	r.Handle("/", chain(getRoot, setRequestID, logRequest)).
		Methods(http.MethodGet)

	r.Handle("/projects", chain(srv.handleCreateProject, setRequestID, logRequest)).
		Methods(http.MethodPost)

	r.Handle("/projects", chain(
		srv.handleGetProjects,
		setRequestID,
		logRequest,
		srv.checkAuth,
	)).Methods(http.MethodGet)

	r.Handle("/projects/{id}", chain(
		srv.handleGetProject,
		setRequestID,
		logRequest,
		srv.checkAuth,
	)).Methods(http.MethodGet)

	// TODO: delete projects

	r.Handle("/projects/{project_id}/git_remotes", chain(
		srv.handleCreateGitRemote,
		setRequestID,
		logRequest,
		srv.checkAuth,
	)).Methods(http.MethodPost)

	r.Handle("/projects/{project_id}/pipelines", chain(
		srv.handleGetPipelines,
		setRequestID,
		logRequest,
		srv.checkAuth,
	)).Methods(http.MethodGet)

	r.Handle("/pipelines/{id}", chain(
		srv.handleGetPipeline,
		setRequestID,
		logRequest,
		srv.checkAuth,
	)).Methods(http.MethodGet)

	r.Handle("/pipelines/{pid}/runs/{count}", chain(
		srv.handleGetRun,
		setRequestID,
		logRequest,
		srv.checkAuth,
	)).Methods(http.MethodGet)

	r.Handle("/steps/{id}", chain(
		srv.handleGetStep,
		setRequestID,
		logRequest,
		srv.checkAuth,
	)).Methods(http.MethodGet)

	r.Handle("/tasks/{id}", chain(
		srv.handleGetTask,
		setRequestID,
		logRequest,
		srv.checkAuth,
	)).Methods(http.MethodGet)

	r.Handle("/auth", chain(srv.handleAuth, setRequestID, logRequest)).
		Methods(http.MethodPost)

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

func (srv *Server) checkAuth(f http.HandlerFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		hdrline, ok := req.Header["Authorization"]
		if !ok {
			err := errors.New("missing bearer token")

			logger.WithError(err).Error("unable to authorize request")
			writeErrResp(rw, err, http.StatusUnauthorized)
			return
		}

		hdr := strings.Split(hdrline[0], " ")

		if len(hdr) < 2 {
			err := errors.New("missing bearer token")

			logger.WithError(err).Error("unable to authorize request")
			writeErrResp(rw, err, http.StatusUnauthorized)
			return
		}

		// Tokens come in the form of "Bearer $TOKEN"
		bearer := hdr[1]

		keyfn := func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				err := errors.New("invalid signing method for bearer token")

				return nil, err
			}

			return srv.jwtsecret, nil
		}

		token, err := jwt.ParseWithClaims(bearer, &jwt.StandardClaims{}, keyfn)
		if err != nil {
			logger.WithError(err).Error("unable to authorize request")
			writeErrResp(rw, err, http.StatusUnauthorized)
			return
		}

		if claims, ok := token.Claims.(*jwt.StandardClaims); ok && token.Valid {
			if time.Now().Unix() > claims.ExpiresAt {
				err := errors.New("token expired")
				logger.WithError(err).Error("unable to authorize request")
				writeErrResp(rw, err, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(req.Context(), keyReqSub, claims.Subject)
			logger.WithField("sub", claims.Subject).
				Debug("setting auth subject")

			f(rw, req.WithContext(ctx))
			return
		}

		err = errors.New("invalid bearer token")
		logger.WithError(err).Error("unable to authorize request")
		writeErrResp(rw, err, http.StatusUnauthorized)
		return
	}
}
