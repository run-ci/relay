package http

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

func (st *memStore) Authenticate(user, pass string) error {
	return nil
}

func TestCheckAuth(t *testing.T) {
	testfn := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		sub := req.Context().Value(keyReqSub).(string)
		respbody := map[string]string{
			"sub": sub,
		}

		buf, _ := json.Marshal(respbody)
		rw.Write(buf)
		return
	})

	jwtsecret := []byte("valid")
	srv := &Server{
		jwtsecret: jwtsecret,
	}

	handler := srv.checkAuth(testfn)

	// RETURNING UNAUTHORIZED
	// request with no authorization header
	reqNoHeader := httptest.NewRequest("", "http://test", nil)

	// request with invalid format "$VALIDTOKEN"
	reqNoBearer := httptest.NewRequest("", "http://test", nil)
	token := gentoken(time.Now().Add(15*time.Minute), jwtsecret, jwt.SigningMethodHS256)
	reqNoBearer.Header["Authorization"] = []string{token}

	// request with format "Bearer $INVALID_TOKEN"
	reqInvalidBearer := httptest.NewRequest("", "http://test", nil)
	reqInvalidBearer.Header["Authorization"] = []string{"Bearer", "INVALID"}

	// TODO: figure out how to test an invalid signature
	// request with invalid signing method for token
	//reqInvalidSignature := httptest.NewRequest("", "http://test", nil)
	//token = gentoken(time.Now().Add(15*time.Minute), jwtsecret, jwt.SigningMethodHS384)
	//reqInvalidSignature.Header["Authorization"] = []string{"Bearer", token}

	// request with different signature
	reqBadSignature := httptest.NewRequest("", "http://test", nil)
	token = gentoken(time.Now().Add(15*time.Minute), []byte("bad"), jwt.SigningMethodHS256)
	reqBadSignature.Header["Authorization"] = []string{"Bearer", token}

	// request with expired token
	reqExpiredToken := httptest.NewRequest("", "http://test", nil)
	token = gentoken(time.Now().Add(-1*time.Minute), jwtsecret, jwt.SigningMethodHS256)
	reqExpiredToken.Header["Authorization"] = []string{"Bearer", token}

	// RETURNING AUTHORIZED
	// request with valid token
	reqValidToken := httptest.NewRequest("", "http://test", nil)
	token = gentoken(time.Now().Add(15*time.Minute), jwtsecret, jwt.SigningMethodHS256)
	reqValidToken.Header["Authorization"] = []string{"Bearer", token}

	type result struct {
		status int
		body   map[string]string
	}

	tests := []struct {
		label    string
		req      *http.Request
		expected result
		actual   result
	}{
		{
			label: "no header",
			req:   reqNoHeader,
			expected: result{
				status: http.StatusUnauthorized,
				body: map[string]string{
					"error": "missing bearer token",
				},
			},
			actual: result{},
		},
		{
			label: "no bearer",
			req:   reqNoBearer,
			expected: result{
				status: http.StatusUnauthorized,
				body: map[string]string{
					"error": "missing bearer token",
				},
			},
			actual: result{},
		},
		{
			label: "invalid bearer",
			req:   reqInvalidBearer,
			expected: result{
				status: http.StatusUnauthorized,
				body: map[string]string{
					"error": "token contains an invalid number of segments",
				},
			},
			actual: result{},
		},
		// TODO: figure out how to test an invalid signature
		//{
		//label: "invalid signature",
		//req:   reqInvalidSignature,
		//expected: result{
		//status: http.StatusUnauthorized,
		//body: map[string]string{
		//"error": "invalid bearer token",
		//},
		//},
		//actual: result{},
		//},
		{
			label: "bad signature",
			req:   reqBadSignature,
			expected: result{
				status: http.StatusUnauthorized,
				body: map[string]string{
					"error": "signature is invalid",
				},
			},
			actual: result{},
		},
		{
			label: "expired token",
			req:   reqExpiredToken,
			expected: result{
				status: http.StatusUnauthorized,
				body: map[string]string{
					"error": "token is expired by 1m0s",
				},
			},
			actual: result{},
		},
		{
			label: "valid token",
			req:   reqValidToken,
			expected: result{
				status: http.StatusOK,
				body: map[string]string{
					"sub": "user@test",
				},
			},
			actual: result{},
		},
	}

	for _, test := range tests {
		t.Run(test.label, func(t *testing.T) {
			rw := httptest.NewRecorder()

			handler(rw, test.req)
			resp := rw.Result()

			test.actual.status = resp.StatusCode
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("got error reading response body: %v", err)
			}

			err = json.Unmarshal(body, &test.actual.body)
			if err != nil {
				t.Fatalf("got error unmarshaling response body: %v", err)
			}

			if test.expected.status != test.actual.status {
				t.Fatalf("expected status %v, got %v",
					test.expected.status,
					test.actual.status,
				)
			}

			if len(test.expected.body) != len(test.actual.body) {
				t.Fatalf("expected body: %+v\n\ngot: %+v\n\n",
					test.expected.body,
					test.actual.body,
				)
			}

			for k, v := range test.expected.body {
				actual, ok := test.actual.body[k]
				if !ok {
					t.Fatalf("expected body: %+v\n\ngot: %+v\n\n",
						test.expected.body,
						test.actual.body,
					)
				}

				if v != actual {
					t.Fatalf("expected body: %+v\n\ngot: %+v\n\n",
						test.expected.body,
						test.actual.body,
					)
				}

			}
		})
	}
}

func gentoken(exp time.Time, secret []byte, signMethod jwt.SigningMethod) string {
	claims := &jwt.StandardClaims{
		ExpiresAt: exp.Unix(),
		Subject:   "user@test",
	}

	token := jwt.NewWithClaims(signMethod, claims)
	ss, _ := token.SignedString(secret)

	return ss
}
