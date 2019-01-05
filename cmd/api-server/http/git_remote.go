package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

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

	// TODO: make sure this is implemented properly since it's been copied
	// for _, remote := range proj.GitRemotes {
	// 	msg := map[string]string{
	// 		"op":     "create",
	// 		"remote": remote.URL,
	// 		"branch": remote.Branch,
	// 	}
	// 	rawmsg, err := json.Marshal(msg)
	// 	if err != nil {
	// 		logger.WithField("error", err).
	// 			Warn("unable to marshal poller create message")
	// 	} else {
	// 		// Not being able to send to the poller is not enough to cause the
	// 		// request to fail. For this reason, we should try as hard as possible
	// 		// to send the request.
	// 		go sendWithBackoff(logger, srv.pollch, rawmsg)
	// 	}
	// }

	rw.WriteHeader(http.StatusCreated)
	return
}
