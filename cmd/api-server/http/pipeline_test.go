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

func (st *memStore) GetPipelines(project int) ([]store.Pipeline, error) {
	pipelines := []store.Pipeline{}
	for _, pipeline := range st.pipelinedb {
		if project == pipeline.ProjectID {
			pipelines = append(pipelines, pipeline)
		}
	}

	return pipelines, nil
}

func (st *memStore) GetPipeline(id int) (store.Pipeline, error) {
	p, ok := st.pipelinedb[id]
	if !ok {
		return p, store.ErrPipelineNotFound
	}

	return p, nil
}

func (st *memStore) seedPipelines() {
	data := []struct {
		id           int
		name         string
		success      bool
		remoteURL    string
		remoteBranch string
		projectID    int
	}{
		{
			id:           1,
			name:         "default",
			success:      true,
			remoteURL:    "https://github.com/run-ci/relay.git",
			remoteBranch: "master",
			projectID:    1,
		},
		{
			id:           2,
			name:         "docker",
			success:      true,
			remoteURL:    "https://github.com/run-ci/relay.git",
			remoteBranch: "master",
			projectID:    1,
		},
		{
			id:           3,
			name:         "default",
			success:      false,
			remoteURL:    "https://github.com/run-ci/run.git",
			remoteBranch: "master",
			projectID:    2,
		},
	}

	for _, d := range data {
		st.pipelinedb[d.projectID] = store.Pipeline{
			ID:      d.id,
			Name:    d.name,
			Success: &d.success,
			GitRemote: store.GitRemote{
				Branch: d.remoteBranch,
				URL:    d.remoteURL,
			},
		}
	}
}

func TestGetPipelines(t *testing.T) {
	st := &memStore{
		pipelinedb: make(map[int]store.Pipeline),
	}
	st.seedPipelines()

	srv := NewServer(":9001", make(chan []byte), st)

	test := struct {
		input    int
		expected []store.Pipeline
		actual   []store.Pipeline
	}{
		input:    0,
		expected: []store.Pipeline{st.pipelinedb[0], st.pipelinedb[1]},
		actual:   []store.Pipeline{},
	}

	r := mux.NewRouter()
	r.Handle("/pipelines/{project_id}", chain(srv.handleGetPipelines, setRequestID))

	ts := httptest.NewServer(r)
	defer ts.Close()

	requrl := fmt.Sprintf("%v/pipelines/%v", ts.URL, test.input)
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
		t.Fatalf("got error unmarshaling pipelines: %v", err)
	}

	if len(test.expected) != len(test.actual) {
		t.Fatalf("expected %v pipelines, got %v", len(test.expected), len(test.actual))
	}

	// TODO: test pipelines that are returned
}

func TestGetPipeline(t *testing.T) {
	st := &memStore{
		pipelinedb: make(map[int]store.Pipeline),
	}
	st.seedPipelines()

	srv := NewServer(":9001", make(chan []byte), st)

	test := struct {
		input    int
		expected store.Pipeline
		actual   store.Pipeline
	}{
		input:    1,
		expected: st.pipelinedb[1],
		actual:   store.Pipeline{},
	}

	r := mux.NewRouter()
	r.Handle("/pipelines/{id}", chain(srv.handleGetPipeline, setRequestID))

	ts := httptest.NewServer(r)
	defer ts.Close()

	requrl := fmt.Sprintf("%v/pipelines/%v", ts.URL, test.input)
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

	if test.expected.ID != test.actual.ID {
		t.Fatalf("expected id %v, got %v", test.expected.ID, test.actual.ID)
	}

	if test.expected.Name != test.actual.Name {
		t.Fatalf("expected Name %v, got %v", test.expected.Name, test.actual.Name)
	}

	if *test.expected.Success != *test.actual.Success {
		t.Fatalf("expected Success %v, got %v", *test.expected.Success, *test.actual.Success)
	}

	if test.expected.ProjectID != test.actual.ProjectID {
		t.Fatalf("expected ProjectID %v, got %v", test.expected.ProjectID, test.actual.ProjectID)
	}

	if test.expected.GitRemote.URL != test.actual.GitRemote.URL {
		t.Fatalf("expected GitRemote.URL %v, got %v", test.expected.GitRemote.URL, test.actual.GitRemote.URL)
	}

	if test.expected.GitRemote.Branch != test.actual.GitRemote.Branch {
		t.Fatalf("expected GitRemote.Branch %v, got %v", test.expected.GitRemote.Branch, test.actual.GitRemote.Branch)
	}

	// TODO: test runs

}

// TODO: test pipeline not found returns 404
