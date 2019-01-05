package http

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/run-ci/relay/store"
	"github.com/sirupsen/logrus"
)

func (srv *Server) handleCreateGitRemote(rw http.ResponseWriter, req *http.Request) {
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

	logger.Debug("checking mux vars for id")
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

	id, err := strconv.Atoi(raw)
	if err != nil {
		logger.WithError(err).Error("unable to parse project id as integer")

		writeErrResp(rw, err, http.StatusBadRequest)
		return
	}

	logger = logger.WithField("project_id", id)

	logger.Debug("unmarshaling request body")
	gr := store.GitRemote{ProjectID: id}
	err = json.Unmarshal(buf, &gr)
	if err != nil {
		logger.WithField("error", err).
			Error("unable to unmarshal request body")

		writeErrResp(rw, err, http.StatusBadRequest)
		return
	}

	logger = logger.WithFields(logrus.Fields{
		"git_remote": fmt.Sprintf("%v#%v", gr.URL, gr.Branch),
	})

	logger.Info("saving git remote")
	err = srv.st.CreateGitRemote(reqSub, &gr)
	if err != nil {
		logger.WithField("error", err).
			Error("unable to save git repo in database")

		writeErrResp(rw, err, http.StatusInternalServerError)
		return
	}

	msg := map[string]string{
		"op":     "create",
		"remote": gr.URL,
		"branch": gr.Branch,
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

	rw.WriteHeader(http.StatusAccepted)
	return
}

func (srv *Server) handleGetGitRemote(rw http.ResponseWriter, req *http.Request) {
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
	if raw, ok = vars["project_id"]; !ok || raw == "" {
		err := errors.New("missing paramter 'project_id' from request")
		logger.WithError(err).Error("unable to complete request")

		writeErrResp(rw, err, http.StatusBadRequest)
		return
	}

	logger.Debug("parsing project id")

	id, err := strconv.Atoi(raw)
	if err != nil {
		logger.WithError(err).Error("unable to parse project id as integer")

		writeErrResp(rw, err, http.StatusBadRequest)
		return
	}

	logger = logger.WithField("project_id", id)

	if raw, ok = vars["id"]; !ok || raw == "" {
		err := errors.New("missing paramter 'id' from request")
		logger.WithError(err).Error("unable to complete request")

		writeErrResp(rw, err, http.StatusBadRequest)
		return
	}

	logger.Debug("decoding git remote id")

	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		logger.WithError(err).Error("unable to decode git remote")

		writeErrResp(rw, err, http.StatusInternalServerError)
		return
	}

	spec := strings.Split(string(decoded), "#")

	logger = logger.WithFields(logrus.Fields{
		"url":    spec[0],
		"branch": spec[1],
	})

	logger.Debug("fetching git remote")

	remote, err := srv.st.GetGitRemote(reqSub, id, spec[0], spec[1])
	if err != nil {
		logger.WithError(err).Error("unable to retrieve git remote")

		writeErrResp(rw, err, http.StatusInternalServerError)
		return
	}

	logger.Debug("marshaling response body")

	buf, err := json.Marshal(remote)
	if err != nil {
		logger.WithError(err).Error("unable to marshal response body")

		writeErrResp(rw, err, http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusOK)
	rw.Write(buf)
	return
}
