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

func (st *memStore) GetStep(id int) (store.Step, error) {
	s, ok := st.stepdb[id]
	if !ok {
		return store.Step{}, store.ErrStepNotFound
	}

	return s, nil
}

func (st *memStore) seedSteps() {
	data := []struct {
		id      int
		name    string
		success bool
	}{
		{
			id:      1,
			name:    "default",
			success: true,
		},
		{
			id:      2,
			name:    "docker",
			success: true,
		},
		{
			id:      3,
			name:    "default",
			success: false,
		},
	}

	for _, d := range data {
		st.stepdb[d.id] = store.Step{
			ID:      d.id,
			Name:    d.name,
			Success: &d.success,
		}
	}
}

func TestGetStep(t *testing.T) {
	st := &memStore{
		stepdb: make(map[int]store.Step),
	}
	st.seedSteps()

	srv := NewServer(":9001", make(chan []byte), st, "test")

	// TODO: test a 404
	test := struct {
		input    int
		expected store.Step
		actual   store.Step
	}{
		input:    1,
		expected: st.stepdb[1],
		actual:   store.Step{},
	}

	r := mux.NewRouter()
	r.Handle("/steps/{id}", chain(srv.handleGetStep, setRequestID))

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

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status code %v, got %v", http.StatusOK, resp.StatusCode)
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

	// TODO: test tasks

}
