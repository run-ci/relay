package http

import "github.com/run-ci/relay/store"

func (st *memStore) GetPipelines(remote string) (pipelines []store.Pipeline, err error) {
	// pipelines := []store.Pipeline{}
	// for _, pipeline := range st.pipelinedb {
	// 	if remote == "" || remote == pipeline.Remote {
	// 		pipelines = append(pipelines, pipeline)
	// 	}
	// }

	return pipelines, err
}

// func (st *memStore) ReadPipeline(p *store.Pipeline) error {

// 	return nil
// }

// func (st *memStore) seedPipelines() {
// 	data := []struct {
// 		remote  string
// 		name    string
// 		success bool
// 	}{
// 		{
// 			"https://github.com/run-ci/relay.git",
// 			"default",
// 			true,
// 		},

// 		{
// 			"https://github.com/run-ci/run.git",
// 			"default",
// 			true,
// 		},
// 	}

// 	for _, d := range data {
// 		remote := d.remote
// 		name := d.name
// 		success := d.success

// 		pipeline := store.Pipeline{
// 			Remote: remote,
// 			Name:   name,
// 			Ref:    "master",
// 			Runs: []store.Run{
// 				store.Run{
// 					PipelineRemote: remote,
// 					PipelineName:   name,

// 					Count:   0,
// 					Success: &success,
// 					Steps: []store.Step{
// 						store.Step{
// 							PipelineRemote: remote,
// 							PipelineName:   name,
// 							RunCount:       0,

// 							Name:    "test",
// 							Success: &success,
// 							Tasks: []store.Task{
// 								store.Task{
// 									StepID: 0,

// 									ID:      0,
// 									Name:    "test",
// 									Success: &success,
// 								},
// 							},
// 						},
// 					},
// 				},
// 			},
// 		}

// 		for _, run := range pipeline.Runs {
// 			run.SetStart()

// 			for _, step := range run.Steps {
// 				step.SetStart()

// 				for _, task := range step.Tasks {
// 					task.SetStart()
// 					task.SetEnd()
// 				}

// 				step.SetEnd()
// 			}

// 			run.SetEnd()
// 		}

// 		st.pipelinedb[remote+name] = pipeline
// 	}
// }

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
