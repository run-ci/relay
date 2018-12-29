package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

func (srv *Server) handleGetStep(rw http.ResponseWriter, req *http.Request) {
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

	logger.Debug("retrieving step from store")

	step, err := srv.st.GetStep(reqSub, id)
	if err != nil {
		logger.WithError(err).Error("unable to retrieve step")

		writeErrResp(rw, err, http.StatusInternalServerError)
		return
	}

	logger.Debug("marshaling response body")

	buf, err := json.Marshal(step)
	if err != nil {
		logger.WithError(err).Error("unable to marshal response body")

		writeErrResp(rw, err, http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusOK)
	rw.Write(buf)
	return
}
