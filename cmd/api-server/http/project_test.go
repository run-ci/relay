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

	"github.com/gorilla/mux"
	"github.com/run-ci/relay/store"
)

type memStore struct {
	projectdb  map[int]store.Project
	pipelinedb map[int]store.Pipeline
	stepdb     map[int]store.Step
	taskdb     map[int]store.Task

	createProject func(proj *store.Project) error
}

func (st *memStore) CreateProject(proj *store.Project) error {
	if st.createProject != nil {
		return st.createProject(proj)
	}

	st.projectdb[proj.ID] = *proj
	return nil
}

func (st *memStore) GetProject(user string, id int) (store.Project, error) {
	ret := st.projectdb[id]
	if ret.User.Email != user {
		return store.Project{}, store.ErrProjectNotFound
	}
	return st.projectdb[id], nil
}

func (st *memStore) GetProjects(user string) ([]store.Project, error) {
	ret := []store.Project{}
	for _, proj := range st.projectdb {
		if proj.User.Email == user {
			ret = append(ret, proj)
		}
	}

	return ret, nil
}

func (st *memStore) seedProjects() {
	st.projectdb[0] = store.Project{
		ID:          0,
		Name:        "test-a",
		Description: "A project used for testing.",
		Authorization: store.Authorization{
			User: store.User{
				Email: "user@test",
			},
		},
	}

	st.projectdb[1] = store.Project{
		ID:          1,
		Name:        "test-b",
		Description: "A project used for testing.",
		Authorization: store.Authorization{
			User: store.User{
				Email: "user@test",
			},
		},
	}
}

func TestPostProject(t *testing.T) {
	send := make(chan []byte)
	st := &memStore{
		projectdb: make(map[int]store.Project),
	}

	// Setting this so that the ID gets set appropriately.
	st.createProject = func(proj *store.Project) error {
		proj.ID = 999
		st.projectdb[proj.ID] = *proj

		return nil
	}

	srv := NewServer(":9001", send, st, "test")

	proj := map[string]string{
		"name":        "test-create-project",
		"description": "A project for testing creation.",
	}

	payload, err := json.Marshal(proj)
	if err != nil {
		t.Fatalf("got error when marshaling request payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "http://test/projects", bytes.NewBuffer(payload))
	req = req.WithContext(context.WithValue(context.Background(), keyReqID, "test"))
	rw := httptest.NewRecorder()

	srv.handleCreateProject(rw, req)

	resp := rw.Result()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected status %v, got %v", http.StatusAccepted, resp.StatusCode)
	}

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("got error reading response body: %v", err)
	}
	defer resp.Body.Close()

	var result store.Project
	err = json.Unmarshal(buf, &result)
	if err != nil {
		t.Fatalf("got error unmarshaling JSON response body: %v", err)
	}

	if result.ID != 999 {
		t.Fatalf("expected project ID to be set to 999, got %v", result.ID)
	}

	if result.Name != proj["name"] {
		t.Fatalf("expected name: %v, got %v", proj["name"], result.Name)
	}

	if result.Description != proj["description"] {
		t.Fatalf("expected description: %v, got %v", proj["description"], result.Description)
	}
}

func TestGetAllProjects(t *testing.T) {
	st := &memStore{
		projectdb: make(map[int]store.Project),
	}
	st.seedProjects()

	srv := NewServer(":9001", make(chan []byte), st, "test")

	req := httptest.NewRequest(http.MethodGet, "http://test/projects", nil)
	ctx := context.WithValue(
		context.WithValue(context.Background(), keyReqID, "test"),
		keyReqSub,
		"user@test",
	)
	req = req.WithContext(ctx)
	rw := httptest.NewRecorder()

	srv.handleGetProjects(rw, req)

	resp := rw.Result()
	payload, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("got error reading response body: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status code %v, got %v", http.StatusOK, resp.StatusCode)
	}

	results := []store.Project{}
	err = json.Unmarshal(payload, &results)
	if err != nil {
		t.Fatalf("got error unmarshaling response body: %v", err)
	}

	if len(results) != len(st.projectdb) {
		t.Fatalf("expected to get %v projects, got %v", len(st.projectdb), len(results))
	}

	for _, result := range results {
		if stored, ok := st.projectdb[result.ID]; !ok {
			t.Fatalf("got repo %+v that isn't in DB", result)
		} else {
			if result.Name != stored.Name {
				t.Fatalf("expected %+v, got %+v", stored, result)
			}

			if result.Description != stored.Description {
				t.Fatalf("expected %+v, got %+v", stored, result)
			}
		}
	}
}

func autoAuth(fn http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		ctx := context.WithValue(
			req.Context(),
			keyReqSub,
			"user@test",
		)
		req = req.WithContext(ctx)

		fn(rw, req)
	})
}

func TestGetProject(t *testing.T) {
	st := &memStore{
		projectdb: make(map[int]store.Project),
	}
	st.seedProjects()

	srv := NewServer(":9001", make(chan []byte), st, "test")

	test := struct {
		input    int
		expected store.Project
		actual   store.Project
	}{
		input:    0,
		expected: st.projectdb[0],
	}

	r := mux.NewRouter()
	r.Handle("/projects/{id}", chain(srv.handleGetProject, setRequestID, autoAuth))

	ts := httptest.NewServer(r)
	defer ts.Close()

	requrl := fmt.Sprintf("%v/projects/%v", ts.URL, test.input)
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
		t.Fatalf("got error unmarshaling JSON response body: %v", err)
	}

	if test.actual.ID != test.expected.ID {
		t.Fatalf("expected project ID to be set to %v, got %v", test.expected.ID, test.actual.ID)
	}

	if test.actual.Name != test.expected.Name {
		t.Fatalf("expected name: %v, got %v", test.expected.Name, test.actual.Name)
	}

	if test.actual.Description != test.expected.Description {
		t.Fatalf("expected description: %v, got %v", test.expected.Description, test.actual.Description)
	}
}

// TODO: test respect of authorization when getting projects

// TODO: rest respect of authorization when getting a single project

// TODO: move this to the test for creating a remote
// func TestPostGitRepo(t *testing.T) {

// 	// This will time out if the request to create the poller wasn't sent. This
// 	// timeout should fail the test.
// 	rawmsg := <-send
// 	plrmsg := map[string]string{}
// 	err = json.Unmarshal(rawmsg, &plrmsg)
// 	if err != nil {
// 		t.Fatalf("got error unmarshalling poller message: %v", err)
// 	}

// 	if op, ok := plrmsg["op"]; !ok || op != "create" {
// 		t.Fatalf(`expected "op" to be set to "create", got %v`, op)
// 	}

// 	if remote, ok := plrmsg["remote"]; !ok || remote != "test" {
// 		t.Fatalf(`expected "remote" to be set to "test", got %v`, remote)
// 	}

// 	if branch, ok := plrmsg["branch"]; !ok || branch != "master" {
// 		t.Fatalf(`expected "branch" to be set to "master", got %v`, branch)
// 	}
// }
