package http

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/run-ci/relay/store"
	"github.com/sirupsen/logrus"
)

func (srv *Server) handleCreateProject(rw http.ResponseWriter, req *http.Request) {
	reqID := req.Context().Value(keyReqID).(string)
	reqSub := req.Context().Value(keyReqSub).(string)
	logger := logger.WithFields(logrus.Fields{
		"request_id":      reqID,
		"request_subject": reqSub,
	})

	logger.Debug("reading request body")
	buf, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logger.WithField("error", err).
			Error("unable to read request body")

		writeErrResp(rw, err, http.StatusInternalServerError)
		return
	}

	logger.Debug("unmarshaling request body")
	var proj store.Project
	err = json.Unmarshal(buf, &proj)
	if err != nil {
		logger.WithField("error", err).
			Error("unable to unmarshal request body")

		writeErrResp(rw, err, http.StatusBadRequest)
		return
	}

	logger = logger.WithFields(logrus.Fields{
		"project_id": proj.ID,
	})

	proj.User.Email = reqSub

	logger.Info("saving project")
	err = srv.st.CreateProject(&proj)
	if err != nil {
		logger.WithField("error", err).
			Error("unable to save git repo in database")

		writeErrResp(rw, err, http.StatusInternalServerError)
		return
	}

	buf, err = json.Marshal(proj)
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

func (srv *Server) handleGetProjects(rw http.ResponseWriter, req *http.Request) {
	reqID := req.Context().Value(keyReqID).(string)
	reqSub := req.Context().Value(keyReqSub).(string)
	logger := logger.WithFields(logrus.Fields{
		"request_id":      reqID,
		"request_subject": reqSub,
	})

	logger.Debug("retrieving projects from database")

	projects, err := srv.st.GetProjects(reqSub)
	if err != nil {
		logger.WithError(err).Error("unable to retrieve projects from database")

		writeErrResp(rw, err, http.StatusInternalServerError)
		return
	}

	buf, err := json.Marshal(projects)
	if err != nil {
		logger.WithError(err).Error("unable to marshal JSON response body")

		writeErrResp(rw, err, http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusOK)
	rw.Write(buf)
}

func (srv *Server) handleGetProject(rw http.ResponseWriter, req *http.Request) {
	reqID := req.Context().Value(keyReqID).(string)
	reqSub := req.Context().Value(keyReqSub).(string)
	logger := logger.WithFields(logrus.Fields{
		"request_id":      reqID,
		"request_subject": reqSub,
	})

	logger.Debug("checking mux vars for id")
	vars := mux.Vars(req)

	var raw string
	var ok bool
	if raw, ok = vars["id"]; !ok || raw == "" {
		err := errors.New("missing paramter 'id' from request")
		logger.WithError(err).Error("unable to complete request")

		writeErrResp(rw, err, http.StatusBadRequest)
		return
	}

	logger.Debug("parsing id")

	id, err := strconv.Atoi(raw)
	if err != nil {
		logger.WithError(err).Error("unable to parse project id as integer")

		writeErrResp(rw, err, http.StatusBadRequest)
		return
	}

	logger = logger.WithField("project_id", id)
	logger.Debug("getting project")

	proj, err := srv.st.GetProject(reqSub, id)
	if err != nil {
		logger.WithError(err).Error("unable to retrieve project from database")

		writeErrResp(rw, err, http.StatusInternalServerError)
		return
	}

	buf, err := json.Marshal(proj)
	if err != nil {
		logger.WithError(err).Error("unable to marshal response body")

		writeErrResp(rw, err, http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusOK)
	rw.Write(buf)
}
