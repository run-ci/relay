package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/run-ci/relay/store"
)

type memStore struct {
	db map[string]store.GitRepo
}

func (st *memStore) CreateGitRepo(repo store.GitRepo) error {
	key := fmt.Sprintf("%v#%v", repo.Remote, repo.Branch)
	st.db[key] = repo

	return nil
}

func (st *memStore) GetGitRepo(remote, branch string) (store.GitRepo, error) {
	key := fmt.Sprintf("%v#%v", remote, branch)

	return st.db[key], nil
}

func (st *memStore) GetGitRepos() ([]store.GitRepo, error) {
	ret := make([]store.GitRepo, len(st.db))
	i := 0
	for _, repo := range st.db {
		ret[i] = repo
		i++
	}

	return ret, nil
}

func (st *memStore) seedRepos() {
	st.db["test.git#master"] = store.GitRepo{
		Remote: "test.git",
		Branch: "master",
	}

	st.db["test.git#feature"] = store.GitRepo{
		Remote: "test.git",
		Branch: "feature",
	}

	st.db["https://github.com/run-ci/relay.git#master"] = store.GitRepo{
		Remote: "https://github.com/run-ci/relay.git",
		Branch: "master",
	}
}

func TestPostGitRepo(t *testing.T) {
	send := make(chan []byte)
	st := &memStore{
		db: make(map[string]store.GitRepo),
	}
	srv := NewServer(":9001", send, st)

	repo := gitRepoRequest{
		Remote: "test",
		Branch: "master",
	}
	payload, err := json.Marshal(repo)
	if err != nil {
		t.Fatalf("got error when marshaling request payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "http://test/repos/git", bytes.NewBuffer(payload))
	req = req.WithContext(context.WithValue(context.Background(), keyReqID, "test"))
	rw := httptest.NewRecorder()

	srv.postGitRepo(rw, req)

	resp := rw.Result()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected status %v, got %v", http.StatusAccepted, resp.StatusCode)
	}

	// This will time out if the request to create the poller wasn't sent. This
	// timeout should fail the test.
	rawmsg := <-send
	plrmsg := map[string]string{}
	err = json.Unmarshal(rawmsg, &plrmsg)
	if err != nil {
		t.Fatalf("got error unmarshalling poller message: %v", err)
	}

	if op, ok := plrmsg["op"]; !ok || op != "create" {
		t.Fatalf(`expected "op" to be set to "create", got %v`, op)
	}

	if remote, ok := plrmsg["remote"]; !ok || remote != "test" {
		t.Fatalf(`expected "remote" to be set to "test", got %v`, remote)
	}

	if branch, ok := plrmsg["branch"]; !ok || branch != "master" {
		t.Fatalf(`expected "branch" to be set to "master", got %v`, branch)
	}
}

func TestGetAllGitRepos(t *testing.T) {
	st := &memStore{
		db: make(map[string]store.GitRepo),
	}
	st.seedRepos()

	srv := NewServer(":9001", make(chan []byte), st)

	req := httptest.NewRequest(http.MethodGet, "http://test/repos/git", nil)
	req = req.WithContext(context.WithValue(context.Background(), keyReqID, "test"))
	rw := httptest.NewRecorder()

	srv.getGitRepo(rw, req)

	resp := rw.Result()
	payload, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("got error reading response body: %v", err)
	}
	defer resp.Body.Close()

	repos := []gitRepoResponse{}
	err = json.Unmarshal(payload, &repos)
	if err != nil {
		t.Fatalf("got error unmarshaling response body: %v", err)
	}

	if len(repos) != len(st.db) {
		t.Fatalf("expected to get %v repos, got %v", len(st.db), len(repos))
	}

	for _, repo := range repos {
		key := fmt.Sprintf("%v#%v", repo.Remote, repo.Branch)
		if _, ok := st.db[key]; !ok {
			t.Fatalf("got repo %v that isn't in DB", key)
		}
	}
}

func TestGetGitRepo(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	st := &memStore{
		db: make(map[string]store.GitRepo),
	}
	st.seedRepos()

	srv := NewServer(":9001", make(chan []byte), st)

	remote := "test.git"
	url := fmt.Sprintf("http://test/repos/git?remote=%v", remote)

	req := httptest.NewRequest(http.MethodGet, url, nil)
	req = req.WithContext(context.WithValue(context.Background(), keyReqID, "test"))
	rw := httptest.NewRecorder()

	srv.getGitRepo(rw, req)

	resp := rw.Result()
	payload, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("got error reading response body: %v", err)
	}
	defer resp.Body.Close()

	repo := gitRepoResponse{}
	err = json.Unmarshal(payload, &repo)
	if err != nil {
		t.Fatalf("got error unmarshaling response body: %v", err)
	}

	t.Logf("got repo %#v", repo)

	if repo.Remote != remote {
		t.Fatalf("expected remote %v, got %v", remote, repo.Remote)
	}

	if repo.Branch != "master" {
		t.Fatalf("expected default branch to be master, got %v", repo.Branch)
	}
}

func TestGetGitRepoWithBranch(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	st := &memStore{
		db: make(map[string]store.GitRepo),
	}
	st.seedRepos()

	srv := NewServer(":9001", make(chan []byte), st)

	remote := "test.git"
	branch := "feature"
	url := fmt.Sprintf("http://test/repos/git?remote=%v&branch=%v", remote, branch)

	req := httptest.NewRequest(http.MethodGet, url, nil)
	req = req.WithContext(context.WithValue(context.Background(), keyReqID, "test"))
	rw := httptest.NewRecorder()

	srv.getGitRepo(rw, req)

	resp := rw.Result()
	payload, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("got error reading response body: %v", err)
	}
	defer resp.Body.Close()

	repo := gitRepoResponse{}
	err = json.Unmarshal(payload, &repo)
	if err != nil {
		t.Fatalf("got error unmarshaling response body: %v", err)
	}

	t.Logf("got repo %#v", repo)

	if repo.Remote != remote {
		t.Fatalf("expected remote %v, got %v", remote, repo.Remote)
	}

	if repo.Branch != branch {
		t.Fatalf("expected branch to be %v, got %v", branch, repo.Branch)
	}
}
