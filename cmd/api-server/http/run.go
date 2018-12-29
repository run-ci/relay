package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

func (srv *Server) handleGetRun(rw http.ResponseWriter, req *http.Request) {
	reqID := req.Context().Value(keyReqID).(string)
	reqSub := req.Context().Value(keyReqSub).(string)
	logger := logger.WithFields(logrus.Fields{
		"request_id":      reqID,
		"request_subject": reqSub,
	})

	logger.Debug("checking mux vars for pipeline id")
	vars := mux.Vars(req)

	var raw string
	var ok bool
	if raw, ok = vars["pid"]; !ok || raw == "" {
		err := errors.New("missing paramter 'pid' from request")
		logger.WithError(err).Error("unable to complete request")

		writeErrResp(rw, err, http.StatusBadRequest)
		return
	}

	logger.Debug("parsing pipeline id")

	pid, err := strconv.Atoi(raw)
	if err != nil {
		logger.WithError(err).Error("unable to parse pid as integer")

		writeErrResp(rw, err, http.StatusBadRequest)
		return
	}

	logger = logger.WithField("pid", pid)
	logger.Debug("checking mux vars for count")

	if raw, ok = vars["count"]; !ok || raw == "" {
		err := errors.New("missing paramter 'count' from request")
		logger.WithError(err).Error("unable to complete request")

		writeErrResp(rw, err, http.StatusBadRequest)
		return
	}

	logger.Debug("parsing count")

	count, err := strconv.Atoi(raw)
	if err != nil {
		logger.WithError(err).Error("unable to parse count as integer")

		writeErrResp(rw, err, http.StatusBadRequest)
		return
	}

	logger = logger.WithField("count", count)

	logger.Debug("retrieving run from store")

	run, err := srv.st.GetRun(reqSub, pid, count)
	if err != nil {
		logger.WithError(err).Error("unable to retrieve run")

		writeErrResp(rw, err, http.StatusInternalServerError)
		return
	}

	logger.Debug("marshaling response body")

	buf, err := json.Marshal(run)
	if err != nil {
		logger.WithError(err).Error("unable to marshal response body")

		writeErrResp(rw, err, http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusOK)
	rw.Write(buf)
	return
}
