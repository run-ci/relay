package http

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

type jwtClaims struct {
	jwt.StandardClaims

	Email string `json:"email"`
}

func (srv *Server) handleAuth(rw http.ResponseWriter, req *http.Request) {
	reqID := req.Context().Value(keyReqID).(string)
	logger := logger.WithField("request_id", reqID)

	buf, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logger.WithError(err).Error("unable to read request body")

		writeErrResp(rw, err, http.StatusInternalServerError)
	}

	var auth map[string]string
	err = json.Unmarshal(buf, &auth)
	if err != nil {
		logger.WithError(err).Error("unable to unmarshal request body")

		writeErrResp(rw, err, http.StatusBadRequest)
	}

	if _, ok := auth["email"]; !ok {
		err := errors.New("missing fields in auth request body")
		logger.WithError(err).Error("unable to authenticate")

		writeErrResp(rw, err, http.StatusBadRequest)
	}

	if _, ok := auth["password"]; !ok {
		err := errors.New("missing fields in auth request body")
		logger.WithError(err).Error("unable to authenticate")

		writeErrResp(rw, err, http.StatusBadRequest)
	}

	err = srv.st.Authenticate(auth["email"], auth["password"])
	if err != nil {
		logger.WithError(err).Error("unable to authenticate")

		writeErrResp(rw, err, http.StatusBadRequest)
		return
	}

	claims := &jwt.StandardClaims{
		ExpiresAt: time.Now().Add(15 * time.Minute).Unix(),
		// TODO: make this a constant or configurable by ENV
		Issuer:  "relay",
		Subject: auth["email"],
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, err := token.SignedString(srv.jwtsecret)
	if err != nil {
		logger.WithError(err).Error("unable to generate token")

		writeErrResp(rw, err, http.StatusInternalServerError)
		return
	}

	buf, err = json.Marshal(map[string]string{
		"token": ss,
	})

	rw.WriteHeader(http.StatusOK)
	rw.Write(buf)
}
