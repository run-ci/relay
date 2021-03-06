package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type pipelineResponse struct {
	Remote string        `json:"remote"`
	Name   string        `json:"name"`
	Ref    string        `json:"ref"`
	Runs   []runResponse `json:"runs"`
}

type runResponse struct {
	Count   int            `json:"count"`
	Start   *time.Time     `json:"start"`
	End     *time.Time     `json:"end"`
	Success bool           `json:"success"`
	Steps   []stepResponse `json:"steps"`
}

type stepResponse struct {
	ID      int            `json:"id"`
	Name    string         `json:"name"`
	Start   *time.Time     `json:"start"`
	End     *time.Time     `json:"end"`
	Success bool           `json:"success"`
	Tasks   []taskResponse `json:"tasks"`
}

type taskResponse struct {
	ID      int        `json:"id"`
	Name    string     `json:"name"`
	Start   *time.Time `json:"start"`
	End     *time.Time `json:"end"`
	Success bool       `json:"success"`
}

func (srv *Server) handleGetPipelines(rw http.ResponseWriter, req *http.Request) {
	reqID := req.Context().Value(keyReqID).(string)
	reqSub := req.Context().Value(keyReqSub).(string)
	logger := logger.WithFields(logrus.Fields{
		"request_id":      reqID,
		"request_subject": reqSub,
	})

	logger.Debug("checking mux vars for project_id")
	vars := mux.Vars(req)

	var raw string
	var ok bool
	if raw, ok = vars["project_id"]; !ok || raw == "" {
		err := errors.New("missing paramter 'project_id' from request")
		logger.WithError(err).Error("unable to complete request")

		writeErrResp(rw, err, http.StatusBadRequest)
		return
	}

	logger.Debug("parsing id")

	pid, err := strconv.Atoi(raw)
	if err != nil {
		logger.WithError(err).Error("unable to parse project id as integer")

		writeErrResp(rw, err, http.StatusBadRequest)
		return
	}

	logger = logger.WithField("project_id", pid)

	logger.Debug("retrieving pipelines from store")

	pipelines, err := srv.st.GetPipelines(reqSub, pid)
	if err != nil {
		logger.WithError(err).Error("unable to retrieve pipelines")

		writeErrResp(rw, err, http.StatusInternalServerError)
		return
	}

	logger.Debug("marshaling response body")

	buf, err := json.Marshal(pipelines)
	if err != nil {
		logger.WithError(err).Error("unable to marshal response body")

		writeErrResp(rw, err, http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusOK)
	rw.Write(buf)
	return
}

func (srv *Server) handleGetPipeline(rw http.ResponseWriter, req *http.Request) {
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
		logger.WithError(err).Error("unable to parse id as integer")

		writeErrResp(rw, err, http.StatusBadRequest)
		return
	}

	logger = logger.WithField("id", id)

	logger.Debug("retrieving pipelines from store")

	pipeline, err := srv.st.GetPipeline(reqSub, id)
	if err != nil {
		logger.WithError(err).Error("unable to retrieve pipeline")

		writeErrResp(rw, err, http.StatusInternalServerError)
		return
	}

	logger.Debug("marshaling response body")

	buf, err := json.Marshal(pipeline)
	if err != nil {
		logger.WithError(err).Error("unable to marshal response body")

		writeErrResp(rw, err, http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusOK)
	rw.Write(buf)
	return
}
