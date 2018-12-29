package http

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/run-ci/relay/store"
)

// TODO: THESE TESTS ARE BAD!! THEY DON'T TEST AUTHORIZATION!
//
// All requests should be scoped to the user, their group, or public projects. Right
// now these tests don't test for that, and they should!

func (st *memStore) GetRun(user string, pid, n int) (store.Run, error) {
	p, ok := st.pipelinedb[pid]
	if !ok {
		return store.Run{}, store.ErrPipelineNotFound
	}

	if len(p.Runs) < (n + 1) {
		return store.Run{}, store.ErrRunNotFound
	}

	return p.Runs[n], nil
}

func TestGetRun(t *testing.T) {
	st := &memStore{
		pipelinedb: make(map[int]store.Pipeline),
	}
	st.seedPipelines()

	srv := NewServer(":9001", make(chan []byte), st, "test")

	test := struct {
		input    int
		expected store.Run
		actual   store.Run
	}{
		input:    1,
		expected: st.pipelinedb[1].Runs[1],
		actual:   store.Run{},
	}

	r := mux.NewRouter()
	r.Handle("/pipelines/{pid}/runs/{count}", chain(srv.handleGetRun, setRequestID, autoAuth))

	ts := httptest.NewServer(r)
	defer ts.Close()

	requrl := fmt.Sprintf("%v/pipelines/%v/runs/%v", ts.URL, test.input, test.input)
	req, err := http.NewRequest(http.MethodGet, requrl, nil)
	if err != nil {
		t.Fatalf("error creating http request for test: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("error executing test against test server: %v", err)
	}

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("got error reading response body: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status code %v, got %v", http.StatusOK, resp.StatusCode)
	}

	err = json.Unmarshal(buf, &test.actual)
	if err != nil {
		t.Fatalf("got error unmarshaling pipeline: %v", err)
	}

	if test.expected.Count != test.actual.Count {
		t.Fatalf("expected count %v, got %v", test.expected.Count, test.actual.Count)
	}

	if *test.expected.Success != *test.actual.Success {
		t.Fatalf("expected Success %v, got %v", test.expected.Success, test.actual.Success)
	}

	if test.expected.PipelineID != test.actual.PipelineID {
		t.Fatalf("expected PipelineID %v, got %v", test.expected.PipelineID, test.actual.PipelineID)
	}

	// TODO: test steps

}

// TODO: test that request authorization is respected
