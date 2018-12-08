package http

import "github.com/run-ci/relay/store"

func (st *memStore) GetRun(pid, n int) (store.Run, error) {
	p, ok := st.pipelinedb[pid]
	if !ok {
		return store.Run{}, store.ErrPipelineNotFound
	}

	if len(p.Runs) < (n + 1) {
		return store.Run{}, store.ErrRunNotFound
	}

	return p.Runs[n], nil
}
