package http

import (
	"bytes"
	"encoding/base64"
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

func (st *memStore) GetGitRemote(user string, pid int, url string, branch string) (store.GitRemote, error) {
	proj, ok := st.projectdb[pid]
	if !ok {
		return store.GitRemote{}, store.ErrProjectNotFound
	}

	for _, remote := range proj.GitRemotes {
		if remote.URL == url && remote.Branch == branch {
			return remote, nil
		}
	}

	return store.GitRemote{}, store.ErrGitRemoteNotFound
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
		"url": "//test-b.git",
		"branch": "staging"
	}`))
	if err != nil {
		t.Fatalf("error creating http request for test: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("error executing test against test server: %v", err)
	}

	t.Logf("response status: %v", resp.StatusCode)

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected status %v, got %v", http.StatusAccepted, resp.StatusCode)
	}
}

func TestGetGitRemote(t *testing.T) {
	st := &memStore{
		projectdb: make(map[int]store.Project),
	}
	st.seedProjects()

	srv := NewServer(":9001", make(chan []byte), st, "test")

	r := mux.NewRouter()
	r.Handle("/projects/{project_id}/git_remotes/{id}", chain(
		srv.handleGetGitRemote, setRequestID, autoAuth))

	ts := httptest.NewServer(r)
	defer ts.Close()

	// TODO: make this table driven
	spec := base64.StdEncoding.EncodeToString([]byte("//test-a.git#master"))
	requrl := fmt.Sprintf("%v/projects/%v/git_remotes/%v", ts.URL, 0, spec)
	req, err := http.NewRequest(http.MethodGet, requrl, nil)
	if err != nil {
		t.Fatalf("error creating http request for test: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("error executing test against test server: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %v, got %v", http.StatusOK, resp.StatusCode)
	}
}
