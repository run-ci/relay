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

// GetPipelines implements the RelayStore interface. It returns a list of all
// pipelines for the project with the given id.
func (st *Postgres) GetPipelines(pid int) ([]Pipeline, error) {
	// TODO: fix bug here. Not returning project or pipeline id.
	sqlq := `
	SELECT p.name, p.remote_url, p.remote_branch, p.success
	FROM pipelines AS p
	WHERE p.project_id = $1;
	`

	logger := logger.WithFields(log.Fields{
		"project_id": pid,
		"query":      "get_pipelines",
	})

	rows, err := st.db.Query(sqlq, pid)
	if err != nil {
		logger.WithError(err).Debug("unable to query postgres for pipelines")
	}

	ps := []Pipeline{}
	for rows.Next() {
		p := Pipeline{}

		err := rows.Scan(&p.Name, &p.GitRemote.URL, &p.GitRemote.Branch, &p.Success)
		if err != nil {
			logger.WithError(err).Debug("unable to scan row")

			return ps, err
		}

		ps = append(ps, p)
	}

	return ps, nil
}

// GetPipeline retrieves the Pipeline with the given id from postgres.
func (st *Postgres) GetPipeline(id int) (Pipeline, error) {
	logger := logger.WithField("id", id)
	logger.Debug("getting pipeline from postgres")

	sqlq := `
	SELECT name, success, remote_url, remote_branch
	FROM pipelines
	WHERE id = $1
	`

	var pipeline Pipeline
	err := st.db.QueryRow(sqlq, id).Scan(
		&pipeline.Name,
		&pipeline.Success,
		&pipeline.GitRemote.URL,
		&pipeline.GitRemote.Branch,
	)
	if err != nil {
		logger.WithError(err).Debug("unable to query database")
	}

	return pipeline, err
}

// UpdatePipeline is part of the RelayStore interface.
func (st *Postgres) UpdatePipeline(p *Pipeline) error {
	// TODO: fix bug here. The finish time is never updated.
	sqlupdate := `
	UPDATE pipelines
	SET success = $1
	WHERE pipelines.id = $2
	`

	logger := logger.WithFields(log.Fields{
		"id":      p.ID,
		"success": p.Success,
		"query":   "set_pipeline_success",
	})

	logger.Debug("setting pipeline success")

	_, err := st.db.Exec(sqlupdate, p.Success, p.ID)
	return err
}

// GetPipelineID queries Postgres for the ID of the pipeline matching the
// filters. If no pipelines are found it returns ErrNoPipelines.
func (st *Postgres) GetPipelineID(remote GitRemote, name string) (id int, err error) {
	logger := logger.WithFields(log.Fields{
		"url":    remote.URL,
		"branch": remote.Branch,
		"name":   name,
		"query":  "get_pipeline_id",
	})

	sqlq := `
	SELECT id
	FROM pipelines
	WHERE remote_url = $1
		AND remote_branch = $2
		AND name = $3;
	`

	logger.Debug("retrieving id from postgres")

	err = st.db.QueryRow(sqlq, remote.URL, remote.Branch, name).Scan(&id)
	if err == sql.ErrNoRows {
		err = ErrNoPipelines
	}

	return
}

// CreatePipeline saves a Pipeline to Postgres.
func (st *Postgres) CreatePipeline(p *Pipeline) error {
	logger := logger.WithFields(log.Fields{
		"name":   p.Name,
		"url":    p.GitRemote.URL,
		"branch": p.GitRemote.Branch,

		"query": "create_pipeline",
	})

	sqlinsert := `
	WITH project_id AS (
		SELECT project_id FROM git_remotes
		WHERE git_remotes.url = $2
			AND git_remotes.branch = $3
	)
	INSERT INTO pipelines(name, remote_url, remote_branch, project_id)
	SELECT $1, $2, $3, project_id   
	FROM project_id
	RETURNING id;
	`

	logger.Debug("saving pipeline")

	// Using QueryRow because the insert is returning "count".
	err := st.db.QueryRow(
		sqlinsert, p.Name, p.GitRemote.URL, p.GitRemote.Branch).
		Scan(&p.ID)
	if err != nil {
		logger.WithField("error", err).Debug("unable to insert pipeline run")
		return err
	}

	logger.Debug("pipeline saved")

	return nil
}

// CreateRun is part of the PipelineStore interface. It creates a new pipeline
// run in the database and sets the count.
func (st *Postgres) CreateRun(r *Run) error {
	logger := logger.WithFields(log.Fields{
		"pipeline_id": r.PipelineID,
	})

	sqlinsert := `
	WITH run_count AS (
		SELECT COUNT(*) from runs
		WHERE runs.pipeline_id = $4
	)
	INSERT INTO runs (count, start_time, end_time, success, pipeline_id)
	SELECT run_count.count+1, $1, $2, $3, $4
	FROM run_count
	RETURNING count
	`

	logger.Debug("saving pipeline run")

	// Using QueryRow because the insert is returning "count".
	err := st.db.QueryRow(
		sqlinsert, r.Start, r.End, r.Success, r.PipelineID).
		Scan(&r.Count)
	if err != nil {
		logger.WithField("error", err).Debug("unable to insert pipeline run")
		return err
	}

	logger.Debug("pipeline run saved")

	return nil
}

// CreateStep is part of the PipelineStore interface. It creates a new run step
// in the database and sets the ID.
func (st *Postgres) CreateStep(s *Step) error {
	logger := logger.WithFields(log.Fields{
		"pipeline_id": s.PipelineID,
		"run_count":   s.RunCount,
		"name":        s.Name,
	})

	sqlinsert := `
	INSERT INTO steps (name, start_time, end_time, success, pipeline_id, run_count)
	VALUES ($1, $2, $3, $4, $5, $6)
	RETURNING id
	`

	logger.Debug("saving run step")

	// Using QueryRow because the insert is returning "id".
	err := st.db.QueryRow(
		sqlinsert, s.Name, s.Start, s.End, s.Success, s.PipelineID, s.RunCount).
		Scan(&s.ID)
	if err != nil {
		logger.WithField("error", err).Debug("unable to insert run step")
		return err
	}

	logger.Debug("run step saved")

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
	logger := logger.WithFields(log.Fields{
		"pipeline_id": r.PipelineID,
		"count":       r.Count,
		"end":         r.End,
		"success":     r.Success,
	})

	sqlupdate := `
	UPDATE runs
	SET success = $1, end_time = $2
	WHERE runs.pipeline_id = $3 AND runs.count = $4
	`

	logger.Debug("saving run step")

	st.db.Exec(sqlupdate, r.Success, r.End, r.PipelineID, r.Count)

	logger.Debug("run step saved")

	return nil
}

// UpdateStep is part of the PipelineStore interface. It update's a step's
// success status and end time with what's passed in.
func (st *Postgres) UpdateStep(s *Step) error {
	logger := logger.WithFields(log.Fields{
		"pipeline_id": s.PipelineID,
		"run_count":   s.RunCount,
		"name":        s.Name,
		"id":          s.ID,
		"success":     s.Success,
		"end":         s.End,
	})

	sqlupdate := `
	UPDATE steps
	SET success = $1, end_time = $2
	WHERE steps.id = $3
	`

	logger.Debug("saving run step")

	st.db.Exec(sqlupdate, s.Success, s.End, s.ID)

	logger.Debug("run step saved")

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
