package http

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/run-ci/relay/store"
	"github.com/sirupsen/logrus"
)

type gitRepoRequest struct {
	Remote string `json:"remote"`
	Branch string `json:"branch"`
}

type gitRepoResponse struct {
	Remote string `json:"remote"`
	Branch string `json:"branch"`
}

func (srv *Server) postGitRepo(rw http.ResponseWriter, req *http.Request) {
	reqID := req.Context().Value(keyReqID).(string)
	logger := logger.WithField("request_id", reqID)

	logger.Debug("reading request body")
	buf, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logger.WithField("error", err).
			Error("unable to read request body")

		writeErrResp(rw, err, http.StatusInternalServerError)
		return
	}

	logger.Debug("unmarshaling request body")
	var repo gitRepoRequest
	err = json.Unmarshal(buf, &repo)
	if err != nil {
		logger.WithField("error", err).
			Error("unable to unmarshal request body")

		writeErrResp(rw, err, http.StatusBadRequest)
		return
	}

	if repo.Branch == "" {
		repo.Branch = "master"
	}

	logger = logger.WithFields(logrus.Fields{
		"remote": repo.Remote,
		"branch": repo.Branch,
	})

	logger.Info("adding git repo")
	err = srv.st.CreateGitRepo(store.GitRepo{
		Remote: repo.Remote,
		Branch: repo.Branch,
	})
	if err != nil {
		logger.WithField("error", err).
			Error("unable to save git repo in database")

		writeErrResp(rw, err, http.StatusInternalServerError)
		return
	}

	msg := map[string]string{
		"op":     "create",
		"remote": repo.Remote,
		"branch": repo.Branch,
	}
	rawmsg, err := json.Marshal(msg)
	if err != nil {
		logger.WithField("error", err).
			Warn("unable to marshal poller create message")
	} else {
		// Not being able to send to the poller is not enough to cause the
		// request to fail. For this reason, we should try as hard as possible
		// to send the request.
		go sendWithBackoff(logger, srv.pollch, rawmsg)
	}

	resp := gitRepoResponse{
		Remote: repo.Remote,
		Branch: repo.Branch,
	}
	buf, err = json.Marshal(resp)
	if err != nil {
		logger.WithField("error", err).
			Error("unable to marshal response body")

		// We've already processed the request and taken action on it,
		// so returning an error response code here would be misleading.
		writeErrResp(rw, err, http.StatusAccepted)
		return
	}

	rw.WriteHeader(http.StatusAccepted)
	rw.Write(buf)
	return
}

func (srv *Server) getGitRepo(rw http.ResponseWriter, req *http.Request) {
	reqID := req.Context().Value(keyReqID).(string)
	logger := logger.WithField("request_id", reqID)

	if _, ok := req.URL.Query()["remote"]; !ok {
		logger.Info("missing 'remote' argument, fetching all repos")

		srv.getAllRepos(logger, rw)
		return
	}
	remote := req.URL.Query()["remote"][0]

	branch := "master"
	if _, ok := req.URL.Query()["branch"]; ok {
		branch = req.URL.Query()["branch"][0]
	}

	logger.Infof("using %v as branch", branch)

	srv.getRepo(remote, branch, logger, rw)
	return
}

func (srv *Server) getAllRepos(logger *logrus.Entry, rw http.ResponseWriter) {
	repos, err := srv.st.GetGitRepos()
	if err != nil {
		logger.WithField("error", err).Error("unable to get git repos from database")

		writeErrResp(rw, err, http.StatusInternalServerError)
		return
	}

	resp := []gitRepoResponse{}
	for _, repo := range repos {
		resp = append(resp, gitRepoResponse{
			Remote: repo.Remote,
			Branch: repo.Branch,
		})
	}

	buf, err := json.Marshal(resp)
	if err != nil {
		logger.WithField("error", err).Error("unable to marshal response body")

		writeErrResp(rw, err, http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusOK)
	rw.Write(buf)
	return
}

func (srv *Server) getRepo(remote, branch string, logger *logrus.Entry, rw http.ResponseWriter) {
	logger = logger.WithFields(logrus.Fields{
		"remote": remote,
		"branch": branch,
	})
	logger.Debug("getting repo")

	repo, err := srv.st.GetGitRepo(remote, branch)
	if err == sql.ErrNoRows {
		logger.WithField("error", err).Error("repo not found in database")

		writeErrResp(rw, errors.New("repo not found"), http.StatusNotFound)
		return
	}
	if err != nil {
		logger.WithField("error", err).Error("unable to fetch repo from database")

		writeErrResp(rw, err, http.StatusInternalServerError)
		return
	}

	resp := gitRepoResponse{
		Remote: repo.Remote,
		Branch: repo.Branch,
	}
	buf, err := json.Marshal(resp)
	if err != nil {
		logger.WithField("error", err).Error("unable to marshal response body")
		writeErrResp(rw, err, http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusOK)
	rw.Write(buf)
	return
}
