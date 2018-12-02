package store

type rootnode struct {
	children map[string]*pipelinenode
}

type pipelinenode struct {
	children map[string]*runnode
	data     Pipeline
}

type runnode struct {
	children map[int]*stepnode
	data     Run
}

type stepnode struct {
	children map[int]*tasknode
	data     Step
}

type tasknode struct {
	data Task
}
