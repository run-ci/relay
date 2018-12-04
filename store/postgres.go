package store

import (
	"database/sql"

	_ "github.com/lib/pq" // load the postgres driver
	log "github.com/sirupsen/logrus"
)

// Postgres is a PostgreSQL database that's also a PipelineStore.
type Postgres struct {
	db *sql.DB
}

// NewPostgres returns a PipelineStore backed by PostgreSQL. It connects to the
// database using connstr.
func NewPostgres(connstr string) (RelayStore, error) {
	logger = logger.WithField("store", "postgres")

	logger.Debug("connecting to database")

	db, err := sql.Open("postgres", connstr)
	if err != nil {
		logger.WithField("error", err).Debug("unable to connect to database")
		return nil, err
	}

	return &Postgres{
		db: db,
	}, nil
}

// CreateProject saves the project in the database and sets its ID to
// what Postgres assigned it.
func (st *Postgres) CreateProject(p *Project) error {
	logger := logger.WithField("project", p.Name)
	logger.Debug("saving project to postgres")

	sqlinsert := `
	INSERT INTO projects (name, description)
	VALUES
		($1, $2)
	RETURNING id;
	`

	// Using QueryRow because the insert is returning "count".
	err := st.db.QueryRow(sqlinsert, p.Name, p.Description).
		Scan(p.ID)

	if err != nil {
		logger.WithField("error", err).
			Debug("unable to create project")
	}
	return err
}

// GetProject retrieves the Project with the given id from postgres.
func (st *Postgres) GetProject(id int) (Project, error) {
	logger := logger.WithField("project_id", id)
	logger.Debug("getting project from postgres")

	sqlq := `
	SELECT proj.id, proj.name, proj.description,
		gr.url, gr.branch
	FROM projects AS proj
	INNER JOIN git_remotes AS gr
	ON proj.id = gr.project_id
	WHERE proj.id = $1;
	`

	rows, err := st.db.Query(sqlq, id)
	if err != nil {
		logger.WithError(err).Debug("unable to query database")
		return Project{}, err
	}

	p := Project{
		GitRemotes: []GitRemote{},
	}
	for rows.Next() {
		var gr GitRemote
		var desc sql.NullString
		// It's safe to always overwrite `p` here because these values
		// should always be the same.
		err := rows.Scan(&p.ID, &p.Name, &desc, &gr.URL, &gr.Branch)
		if err != nil {
			logger.WithError(err).Debug("unable to scan row")
			return p, err
		}

		if desc.Valid {
			p.Description = desc.String
		}

		p.GitRemotes = append(p.GitRemotes, gr)
	}

	return p, nil
}

// GetProjects retrieves all Projects from Postgres.
func (st *Postgres) GetProjects() ([]Project, error) {
	logger.Debug("fetching all projects from postgres")

	sqlq := `
	SELECT id, name, description FROM projects;
	`

	rows, err := st.db.Query(sqlq)
	if err != nil {
		logger.WithField("error", err).Debug("unable to query database")
		return nil, err
	}

	ps := []Project{}
	for rows.Next() {
		p := Project{}
		var desc sql.NullString
		err := rows.Scan(&p.ID, &p.Name, &desc)
		if err != nil {
			logger.WithField("error", err).Debug("unable to scan row")
			return ps, err
		}

		if desc.Valid {
			p.Description = desc.String
		}

		ps = append(ps, p)
	}

	return ps, nil
}

// GetPipelines is part of the PipelineStore interface. It returns
// all pipelines stored in the database, filtered by remote.
func (st *Postgres) GetPipelines(remote GitRemote) ([]Pipeline, error) {
	// logger.WithField("query", "get_pipelines")
	// logger.Debug("loading pipelines")

	// q := `SELECT p.name, p.remote, p.ref,
	// 			r.count, r.start_time, r.end_time, r.success,
	// 			s.id, s.name, s.start_time, s.end_time, s.success,
	// 			t.id, t.name, t.start_time, t.end_time, t.success
	// 		FROM pipelines AS p INNER JOIN runs AS r
	// 			ON p.remote = r.pipeline_remote
	// 			AND p.name = r.pipeline_name
	// 		INNER JOIN steps AS s
	// 			ON s.pipeline_remote = p.remote
	// 			AND s.pipeline_name = p.name
	// 			AND s.run_count = r.count
	// 		INNER JOIN tasks AS t
	// 			ON t.step_id = s.id`

	// var rows *sql.Rows
	// var err error
	// switch {
	// case remote != "":
	// 	q = fmt.Sprintf(`%v
	// 		WHERE p.remote = $1`, q)
	// 	rows, err = st.db.Query(q, remote)

	// default:
	// 	rows, err = st.db.Query(q)
	// }

	// if err != nil {
	// 	logger.WithError(err).Debug("unable to query database")
	// 	return nil, err
	// }

	// root := rootnode{
	// 	children: make(map[string]*pipelinenode),
	// }

	// for rows.Next() {
	// 	pipeline := Pipeline{}
	// 	run := Run{}
	// 	step := Step{}
	// 	task := Task{}

	// 	logger.Debug("scanning row")

	// 	err := rows.Scan(
	// 		&pipeline.Name, &pipeline.Remote, &pipeline.Ref,
	// 		&run.Count, &run.Start, &run.End, &run.Success,
	// 		&step.ID, &step.Name, &step.Start, &step.End, &step.Success,
	// 		&task.ID, &task.Name, &task.Start, &task.End, &task.Success,
	// 	)
	// 	if err != nil {
	// 		logger.WithError(err).Debug("unable to scan row")
	// 		return nil, err
	// 	}

	// 	pkey := fmt.Sprintf("%v%v", pipeline.Remote, pipeline.Name)
	// 	rkey := fmt.Sprintf("%v%v", pkey, run.Count)
	// 	skey := step.ID

	// 	pnode, ok := root.children[pkey]
	// 	if !ok {
	// 		logger.Debug("pipeline cache miss")

	// 		pnode = &pipelinenode{
	// 			children: make(map[string]*runnode),
	// 			data:     pipeline,
	// 		}

	// 		rnode := &runnode{
	// 			children: make(map[int]*stepnode),
	// 			data:     run,
	// 		}

	// 		snode := &stepnode{
	// 			children: make(map[int]*tasknode),
	// 			data:     step,
	// 		}

	// 		tnode := &tasknode{
	// 			data: task,
	// 		}

	// 		snode.children[task.ID] = tnode
	// 		rnode.children[skey] = snode
	// 		pnode.children[rkey] = rnode
	// 		root.children[pkey] = pnode

	// 		continue
	// 	}

	// 	rnode, ok := pnode.children[rkey]
	// 	if !ok {
	// 		logger.Debug("run cache miss")

	// 		rnode = &runnode{
	// 			children: make(map[int]*stepnode),
	// 			data:     run,
	// 		}

	// 		snode := &stepnode{
	// 			children: make(map[int]*tasknode),
	// 			data:     step,
	// 		}

	// 		tnode := &tasknode{
	// 			data: task,
	// 		}

	// 		snode.children[task.ID] = tnode
	// 		rnode.children[skey] = snode
	// 		pnode.children[rkey] = rnode

	// 		continue
	// 	}

	// 	snode, ok := rnode.children[skey]
	// 	if !ok {
	// 		logger.Debug("step cache miss")

	// 		snode = &stepnode{
	// 			children: make(map[int]*tasknode),
	// 			data:     step,
	// 		}

	// 		tnode := &tasknode{
	// 			data: task,
	// 		}

	// 		snode.children[task.ID] = tnode
	// 		rnode.children[skey] = snode

	// 		continue
	// 	}
	// }

	// pipelines := make([]Pipeline, len(root.children))
	// i := 0
	// for _, pnode := range root.children {
	// 	logger.Debugf("processing pipeline %v", i)

	// 	pipeline := pnode.data

	// 	for _, rnode := range pnode.children {
	// 		logger.Debug("processing run")

	// 		run := rnode.data
	// 		for _, snode := range rnode.children {
	// 			logger.Debug("processing step")

	// 			step := snode.data
	// 			for _, tnode := range snode.children {
	// 				logger.Debug("processing task")

	// 				step.Tasks = append(step.Tasks, tnode.data)
	// 			}
	// 			run.Steps = append(run.Steps, step)
	// 		}
	// 		pipeline.Runs = append(pipeline.Runs, run)
	// 	}

	// 	pipelines[i] = pipeline

	// 	i++
	// }

	// return pipelines, nil
	return []Pipeline{}, nil
}

func (st *Postgres) GetPipeline(id int) (Pipeline, error) {
	// logger := logger.WithFields(log.Fields{
	// 	"pipeline_id": id,
	// })

	return Pipeline{}, nil
}

// CreateRun is part of the PipelineStore interface. It creates a new pipeline
// run in the database and sets the count.
func (st *Postgres) CreateRun(r *Run) error {
	// logger := logger.WithFields(log.Fields{
	// 	"pipeline_remote": r.PipelineRemote,
	// 	"pipeline_name":   r.PipelineName,
	// })

	// sqlinsert := `
	// WITH run_count AS (
	// 	SELECT COUNT(*) from runs
	// 	WHERE runs.pipeline_remote = $4 AND runs.pipeline_name = $5
	// )
	// INSERT INTO runs (count, start_time, end_time, success, pipeline_remote, pipeline_name)
	// SELECT run_count.count+1, $1, $2, $3, $4, $5
	// FROM run_count
	// RETURNING count
	// `

	// logger.Debug("saving pipeline run")

	// // Using QueryRow because the insert is returning "count".
	// err := st.db.QueryRow(
	// 	sqlinsert, r.Start, r.End, r.Success, r.PipelineRemote, r.PipelineName).
	// 	Scan(&r.Count)
	// if err != nil {
	// 	logger.WithField("error", err).Debug("unable to insert pipeline run")
	// 	return err
	// }

	// logger.Debug("pipeline run saved")

	return nil
}

// CreateStep is part of the PipelineStore interface. It creates a new run step
// in the database and sets the ID.
func (st *Postgres) CreateStep(s *Step) error {
	// logger := logger.WithFields(log.Fields{
	// 	"pipeline_remote": s.PipelineRemote,
	// 	"pipeline_name":   s.PipelineName,
	// 	"run_count":       s.RunCount,
	// 	"name":            s.Name,
	// })

	// sqlinsert := `
	// INSERT INTO steps (name, start_time, end_time, success, pipeline_remote, pipeline_name, run_count)
	// VALUES ($1, $2, $3, $4, $5, $6, $7)
	// RETURNING id
	// `

	// logger.Debug("saving run step")

	// // Using QueryRow because the insert is returning "id".
	// err := st.db.QueryRow(
	// 	sqlinsert, s.Name, s.Start, s.End, s.Success, s.PipelineRemote, s.PipelineName, s.RunCount).
	// 	Scan(&s.ID)
	// if err != nil {
	// 	logger.WithField("error", err).Debug("unable to insert run step")
	// 	return err
	// }

	// logger.Debug("run step saved")

	return nil
}

// CreateTask is part of the PipelineStore interface. It creates a new task in
// the database and sets the ID.
func (st *Postgres) CreateTask(t *Task) error {
	logger := logger.WithFields(log.Fields{
		"name":    t.Name,
		"step_id": t.StepID,
	})

	sqlinsert := `
	INSERT INTO tasks (name, start_time, end_time, success, step_id)
	VALUES ($1, $2, $3, $4, $5)
	RETURNING id
	`

	logger.Debug("saving step task")

	// Using QueryRow because the insert is returning "id".
	err := st.db.QueryRow(
		sqlinsert, t.Name, t.Start, t.End, t.Success, t.StepID).
		Scan(&t.ID)
	if err != nil {
		logger.WithField("error", err).Debug("unable to insert step task")
		return err
	}

	logger.Debug("step task saved")

	return nil
}

// UpdateRun implements part of PipelineStore. It updates a run task's success
// status and end time.
func (st *Postgres) UpdateRun(r *Run) error {
	// logger := logger.WithFields(log.Fields{
	// 	"pipeline_remote": r.PipelineRemote,
	// 	"pipeline_name":   r.PipelineName,
	// 	"count":           r.Count,
	// 	"end":             r.End,
	// 	"success":         r.Success,
	// })

	// sqlupdate := `
	// UPDATE runs
	// SET success = $1, end_time = $2
	// WHERE runs.pipeline_remote = $3 AND runs.pipeline_name = $4 AND runs.count = $5
	// `

	// logger.Debug("saving run step")

	// st.db.Exec(sqlupdate, r.Success, r.End, r.PipelineRemote, r.PipelineName, r.Count)

	// logger.Debug("run step saved")

	return nil
}

// UpdateStep is part of the PipelineStore interface. It update's a step's
// success status and end time with what's passed in.
func (st *Postgres) UpdateStep(s *Step) error {
	// logger := logger.WithFields(log.Fields{
	// 	"pipeline_remote": s.PipelineRemote,
	// 	"pipeline_name":   s.PipelineName,
	// 	"run_count":       s.RunCount,
	// 	"name":            s.Name,
	// 	"id":              s.ID,
	// 	"success":         s.Success,
	// 	"end":             s.End,
	// })

	// sqlupdate := `
	// UPDATE steps
	// SET success = $1, end_time = $2
	// WHERE steps.id = $3
	// `

	// logger.Debug("saving run step")

	// st.db.Exec(sqlupdate, s.Success, s.End, s.ID)

	// logger.Debug("run step saved")

	return nil
}

// UpdateTask is part of the PipelineStore interface. It updates the task's
// success status and end time with what's passed in.
func (st *Postgres) UpdateTask(t *Task) error {
	logger := logger.WithFields(log.Fields{
		"name":    t.Name,
		"step_id": t.StepID,
		"success": t.Success,
		"id":      t.ID,
		"end":     t.End,
	})

	sqlupdate := `
	UPDATE tasks
	SET success = $1, end_time = $2
	WHERE tasks.id = $3
	`

	logger.Debug("saving step task")

	st.db.Exec(sqlupdate, t.Success, t.End, t.ID)

	logger.Debug("step task saved")

	return nil
}
