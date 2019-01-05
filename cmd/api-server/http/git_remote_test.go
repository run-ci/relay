package http

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/run-ci/relay/store"
)

func (st *memStore) CreateGitRemote(user string, r *store.GitRemote) error {
	proj, ok := st.projectdb[r.ProjectID]
	if !ok {
		return store.ErrProjectNotFound
	}

	proj.GitRemotes = append(proj.GitRemotes, *r)
	return nil
}

func TestCreateGitRemote(t *testing.T) {
	st := &memStore{
		projectdb: make(map[int]store.Project),
	}
	st.seedProjects()

	srv := NewServer(":9001", make(chan []byte), st, "test")

	r := mux.NewRouter()
	r.Handle("/projects/{project_id}/git_remotes", chain(
		srv.handleCreateGitRemote, setRequestID, autoAuth))

	ts := httptest.NewServer(r)
	defer ts.Close()

	// TODO: make this table driven
	requrl := fmt.Sprintf("%v/projects/%v/git_remotes", ts.URL, 1)
	req, err := http.NewRequest(http.MethodPost, requrl, bytes.NewBufferString(`{
		"url": "https://github.com/run-ci/relay.git",
		"branch": "master"
	}`))
	if err != nil {
		t.Fatalf("error creating http request for test: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("error executing test against test server: %v", err)
	}

	t.Logf("response status: %v", resp.StatusCode)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status %v, got %v", http.StatusCreated, resp.StatusCode)
	}
}
