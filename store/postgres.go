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

// CreateGitRepo saves the Git repository in Postgres.
func (st *Postgres) CreateGitRepo(repo GitRepo) error {
	logger.Debugf("creating git repo for %v", repo.Remote)

	sqlinsert := `
	INSERT INTO git_repos (remote, branch)
	VALUES
		($1, $2);
	`

	_, err := st.db.Exec(sqlinsert, repo.Remote, repo.Branch)
	if err != nil {
		logger.WithField("error", err).
			Debugf("unable to create git repo for %v#%v", repo.Remote, repo.Branch)
	}
	return err
}

// GetGitRepo returns the git repo with the given remote and branch.
func (st *Postgres) GetGitRepo(remote, branch string) (GitRepo, error) {
	logger := logger.WithField("remote", remote)
	logger.Debug("getting git repo from postgres")

	sqlq := `
	SELECT * FROM git_repos
	WHERE remote = $1 AND branch = $2;
	`

	var repo GitRepo
	return repo, st.db.QueryRow(sqlq, remote).Scan(&repo.Remote, &repo.Branch)
}

// GetGitRepos returns all repos.
func (st *Postgres) GetGitRepos() ([]GitRepo, error) {
	logger.Debug("getting git repos from postgres")

	sqlq := `
	SELECT * FROM git_repos;
	`

	rows, err := st.db.Query(sqlq)
	if err != nil {
		logger.WithField("error", err).Debug("unable to query database")
		return nil, err
	}

	repos := []GitRepo{}
	for rows.Next() {
		repo := GitRepo{}
		err := rows.Scan(&repo.Remote, &repo.Branch)
		if err != nil {
			logger.WithField("error", err).Debug("unable to scan row")
			return repos, err
		}
		repos = append(repos, repo)
	}

	return repos, nil
}

// ReadPipeline is part of the PipelineStore interface. If the pipeline
// isn't found, it returns ErrPipelineNotFound.
func (st *Postgres) ReadPipeline(p *Pipeline) error {
	logger := logger.WithFields(log.Fields{
		"pipeline": p,
	})

	q := `SELECT remote, ref, name FROM pipelines
	WHERE pipelines.remote = $1 AND pipelines.name = $2`
	logger = logger.WithField("query", q)

	logger.Debug("loading pipeline")

	row := st.db.QueryRow(q, p.Remote, p.Name)

	logger.Debug("scanning rows")

	err := row.Scan(&p.Remote, &p.Ref, &p.Name)
	if err == sql.ErrNoRows {
		return ErrPipelineNotFound
	}

	return err
}

// CreateRun is part of the PipelineStore interface. It creates a new pipeline
// run in the database and sets the count.
func (st *Postgres) CreateRun(r *Run) error {
	logger := logger.WithFields(log.Fields{
		"pipeline_remote": r.PipelineRemote,
		"pipeline_name":   r.PipelineName,
	})

	sqlinsert := `
	WITH run_count AS (
		SELECT COUNT(*) from runs
		WHERE runs.pipeline_remote = $4 AND runs.pipeline_name = $5
	)
	INSERT INTO runs (count, start_time, end_time, success, pipeline_remote, pipeline_name)
	SELECT run_count.count+1, $1, $2, $3, $4, $5
	FROM run_count
	RETURNING count
	`

	logger.Debug("saving pipeline run")

	// Using QueryRow because the insert is returning "count".
	err := st.db.QueryRow(
		sqlinsert, r.Start, r.End, r.Success, r.PipelineRemote, r.PipelineName).
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
		"pipeline_remote": s.PipelineRemote,
		"pipeline_name":   s.PipelineName,
		"run_count":       s.RunCount,
		"name":            s.Name,
	})

	sqlinsert := `
	INSERT INTO steps (name, start_time, end_time, success, pipeline_remote, pipeline_name, run_count)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	RETURNING id
	`

	logger.Debug("saving run step")

	// Using QueryRow because the insert is returning "id".
	err := st.db.QueryRow(
		sqlinsert, s.Name, s.Start, s.End, s.Success, s.PipelineRemote, s.PipelineName, s.RunCount).
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
		"pipeline_remote": r.PipelineRemote,
		"pipeline_name":   r.PipelineName,
		"count":           r.Count,
		"end":             r.End,
		"success":         r.Success,
	})

	sqlupdate := `
	UPDATE runs
	SET success = $1, end_time = $2
	WHERE runs.pipeline_remote = $3 AND runs.pipeline_name = $4 AND runs.count = $5
	`

	logger.Debug("saving run step")

	st.db.Exec(sqlupdate, r.Success, r.End, r.PipelineRemote, r.PipelineName, r.Count)

	logger.Debug("run step saved")

	return nil
}

// UpdateStep is part of the PipelineStore interface. It update's a step's
// success status and end time with what's passed in.
func (st *Postgres) UpdateStep(s *Step) error {
	logger := logger.WithFields(log.Fields{
		"pipeline_remote": s.PipelineRemote,
		"pipeline_name":   s.PipelineName,
		"run_count":       s.RunCount,
		"name":            s.Name,
		"id":              s.ID,
		"success":         s.Success,
		"end":             s.End,
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
