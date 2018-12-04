package http

import (
	"time"
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

// func (srv *Server) getPipelines(rw http.ResponseWriter, req *http.Request) {
// 	reqID := req.Context().Value(keyReqID).(string)
// 	logger := logger.WithField("request_id", reqID)

// 	var remote string

// 	if _, ok := req.URL.Query()["remote"]; ok {
// 		logger.Info("filtering pipelines by 'remote'")

// 		var err error
// 		remote, err = url.QueryUnescape(req.URL.Query()["remote"][0])
// 		if err != nil {
// 			logger.WithError(err).Error("unable to query-unescape remote")

// 			writeErrResp(rw, err, http.StatusBadRequest)
// 			return
// 		}

// 		logger = logger.WithField("remote", remote)
// 	}

// 	// branch := "master"
// 	// if _, ok := req.URL.Query()["branch"]; ok {
// 	// 	logger.Info("missing '")
// 	// 	branch = req.URL.Query()["branch"][0]
// 	// }

// 	// logger.Infof("using %v as branch", branch)

// 	srv.dbGetPipelines(remote, logger, rw)
// 	return
// }

// func (srv *Server) dbGetPipelines(remote string, logger *logrus.Entry, rw http.ResponseWriter) {
// 	pipelines, err := srv.st.GetPipelines(remote)
// 	if err != nil {
// 		logger.WithError(err).Error("unable to get pipelines from database")

// 		writeErrResp(rw, err, http.StatusInternalServerError)
// 		return
// 	}

// 	buf, err := json.Marshal(pipelines)
// 	if err != nil {
// 		logger.WithField("error", err).Error("unable to marshal response body")

// 		writeErrResp(rw, err, http.StatusInternalServerError)
// 		return
// 	}

// 	rw.WriteHeader(http.StatusOK)
// 	rw.Write(buf)
// 	return
// }
