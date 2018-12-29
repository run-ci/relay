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

func (st *memStore) GetTask(user string, id int) (store.Task, error) {
	t, ok := st.taskdb[id]
	if !ok {
		return store.Task{}, store.ErrTaskNotFound
	}

	return t, nil
}

func (st *memStore) seedTasks() {
	data := []struct {
		id      int
		name    string
		success bool
	}{
		{
			id:      1,
			name:    "build",
			success: true,
		},
		{
			id:      2,
			name:    "test",
			success: true,
		},
		{
			id:      3,
			name:    "package",
			success: false,
		},
	}

	for _, d := range data {
		st.taskdb[d.id] = store.Task{
			ID:      d.id,
			Name:    d.name,
			Success: &d.success,
		}
	}
}

func TestGetTask(t *testing.T) {
	st := &memStore{
		taskdb: make(map[int]store.Task),
	}
	st.seedTasks()

	srv := NewServer(":9001", make(chan []byte), st, "test")

	test := struct {
		input    int
		expected store.Task
		actual   store.Task
		status   int
	}{
		input:    1,
		expected: st.taskdb[1],
		actual:   store.Task{},
		status:   http.StatusOK,
	}

	r := mux.NewRouter()
	r.Handle("/steps/{id}", chain(srv.handleGetTask, setRequestID, autoAuth))

	ts := httptest.NewServer(r)
	defer ts.Close()

	requrl := fmt.Sprintf("%v/steps/%v", ts.URL, test.input)
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

	if resp.StatusCode != test.status {
		t.Fatalf("expected status code %v, got %v", test.status, resp.StatusCode)
	}

	err = json.Unmarshal(buf, &test.actual)
	if err != nil {
		t.Fatalf("got error unmarshaling pipeline: %v", err)
	}

	if test.expected.ID != test.actual.ID {
		t.Fatalf("expected ID %v, got %v", test.expected.ID, test.actual.ID)
	}

	if test.expected.Name != test.actual.Name {
		t.Fatalf("expected Name %v, got %v", test.expected.Name, test.actual.Name)
	}

	if *test.expected.Success != *test.actual.Success {
		t.Fatalf("expected Success %v, got %v", test.expected.Success, test.actual.Success)
	}
}

// TODO: test get /tasks/id respects auth
