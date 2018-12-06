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
			id:           0,
			name:         "test-a",
			success:      true,
			remoteURL:    "https://github.com/run-ci/relay.git",
			remoteBranch: "master",
			projectID:    0,
		},
		{
			id:           1,
			name:         "test-b",
			success:      true,
			remoteURL:    "https://github.com/run-ci/relay.git",
			remoteBranch: "feature",
			projectID:    0,
		},
		{
			id:           2,
			name:         "docker",
			success:      false,
			remoteURL:    "https://github.com/run-ci/relay.git",
			remoteBranch: "master",
			projectID:    0,
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
		expected store.Pipeline
		actual   []store.Pipeline
	}{
		input:    0,
		expected: st.pipelinedb[0],
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
}

// func TestGetAllPipelines(t *testing.T) {
// 	st := &memStore{
// 		pipelinedb: make(map[string]store.Pipeline),
// 	}
// 	st.seedPipelines()

// 	srv := NewServer(":9001", make(chan []byte), st)

// 	req := httptest.NewRequest(http.MethodGet, "http://test/pipelines", nil)
// 	req = req.WithContext(context.WithValue(context.Background(), keyReqID, "test"))
// 	rw := httptest.NewRecorder()

// 	srv.getPipelines(rw, req)

// 	resp := rw.Result()
// 	if resp.StatusCode != http.StatusOK {
// 		t.Fatalf("expected stats %v, got %v", http.StatusOK, resp.StatusCode)
// 	}

// 	payload, err := ioutil.ReadAll(resp.Body)
// 	if err != nil {
// 		t.Fatalf("got error reading response body: %v", err)
// 	}
// 	defer resp.Body.Close()

// 	pipelines := []store.Pipeline{}
// 	err = json.Unmarshal(payload, &pipelines)
// 	if err != nil {
// 		t.Fatalf("got error unmarshaling response body: %v", err)
// 	}

// 	if len(pipelines) != len(st.pipelinedb) {
// 		t.Fatalf("expected to get %v pipelines, got %v", len(st.pipelinedb), len(pipelines))
// 	}

// 	for _, pipeline := range pipelines {
// 		key := pipeline.Remote + pipeline.Name
// 		var stored store.Pipeline
// 		var ok bool

// 		if stored, ok = st.pipelinedb[key]; !ok {
// 			t.Fatalf("got repo %v that isn't in DB", key)
// 		}

// 		if stored.Name != pipeline.Name {
// 			t.Fatalf("expected pipeline named %v, got %v", stored.Name, pipeline.Name)
// 		}

// 		if stored.Remote != pipeline.Remote {
// 			t.Fatalf("expected pipeline remote %v, got %v", stored.Remote, pipeline.Remote)
// 		}

// 		if stored.Ref != pipeline.Ref {
// 			t.Fatalf("expected pipeline ref %v, got %v", stored.Ref, pipeline.Ref)
// 		}

// 		if len(pipeline.Runs) != len(stored.Runs) {
// 			t.Fatalf("expected pipeline to have %v runs, got %v", stored.Runs, pipeline.Runs)
// 		}

// 		// TODO: test runs, test steps, test tasks
// 	}
// }

// func TestGetPipelinesByRemote(t *testing.T) {
// 	st := &memStore{
// 		pipelinedb: make(map[string]store.Pipeline),
// 	}
// 	st.seedPipelines()

// 	srv := NewServer(":9001", make(chan []byte), st)

// 	remote := "https://github.com/run-ci/relay.git"
// 	remoteParam := url.QueryEscape(remote)
// 	requrl := fmt.Sprintf("http://test/pipelines?remote=%v", remoteParam)

// 	req := httptest.NewRequest(http.MethodGet, requrl, nil)
// 	req = req.WithContext(context.WithValue(context.Background(), keyReqID, "test"))
// 	rw := httptest.NewRecorder()

// 	srv.getPipelines(rw, req)

// 	resp := rw.Result()
// 	if resp.StatusCode != http.StatusOK {
// 		t.Fatalf("expected stats %v, got %v", http.StatusOK, resp.StatusCode)
// 	}

// 	payload, err := ioutil.ReadAll(resp.Body)
// 	if err != nil {
// 		t.Fatalf("got error reading response body: %v", err)
// 	}
// 	defer resp.Body.Close()

// 	pipelines := []store.Pipeline{}
// 	err = json.Unmarshal(payload, &pipelines)
// 	if err != nil {
// 		t.Fatalf("got error unmarshaling response body: %v", err)
// 	}

// 	expected, _ := st.GetPipelines(remote)

// 	if len(pipelines) != len(expected) {
// 		t.Fatalf("expected to get %v pipelines, got %v", len(st.pipelinedb), len(pipelines))
// 	}

// 	for _, pipeline := range pipelines {
// 		key := pipeline.Remote + pipeline.Name
// 		var stored store.Pipeline
// 		var ok bool

// 		if stored, ok = st.pipelinedb[key]; !ok {
// 			t.Fatalf("got repo %v that isn't in DB", key)
// 		}

// 		if stored.Name != pipeline.Name {
// 			t.Fatalf("expected pipeline named %v, got %v", stored.Name, pipeline.Name)
// 		}

// 		if stored.Remote != pipeline.Remote {
// 			t.Fatalf("expected pipeline remote %v, got %v", stored.Remote, pipeline.Remote)
// 		}

// 		if stored.Ref != pipeline.Ref {
// 			t.Fatalf("expected pipeline ref %v, got %v", stored.Ref, pipeline.Ref)
// 		}

// 		if len(pipeline.Runs) != len(stored.Runs) {
// 			t.Fatalf("expected pipeline to have %v runs, got %v", stored.Runs, pipeline.Runs)
// 		}

// 		// TODO: test runs, test steps, test tasks
// 	}
// }
