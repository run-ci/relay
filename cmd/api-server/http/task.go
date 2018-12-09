package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/run-ci/relay/store"
)

func (srv *Server) handleGetTask(rw http.ResponseWriter, req *http.Request) {
	reqID := req.Context().Value(keyReqID).(string)
	logger := logger.WithField("request_id", reqID)

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

	logger.Debug("retrieving step from store")

	task, err := srv.st.GetTask(id)
	if err != nil {
		logger.WithError(err).Error("unable to retrieve task")
		if err == store.ErrTaskNotFound {
			writeErrResp(rw, err, http.StatusNotFound)
			return
		}

		writeErrResp(rw, err, http.StatusInternalServerError)
		return
	}

	logger.Debug("marshaling response body")

	buf, err := json.Marshal(task)
	if err != nil {
		logger.WithError(err).Error("unable to marshal response body")

		writeErrResp(rw, err, http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusOK)
	rw.Write(buf)
	return
}
