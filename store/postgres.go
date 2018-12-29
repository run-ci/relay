package store

import (
	"database/sql"

	_ "github.com/lib/pq" // load the postgres driver
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
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
	INSERT INTO projects (name, description, user_email, group_name, permissions)
	VALUES
		($1, $2, $3, $4, $5)
	RETURNING id;
	`

	// Using QueryRow because the insert is returning "count".
	err := st.db.QueryRow(sqlinsert, p.Name, p.Description,
		p.User.Email, p.Group.Name, p.Permissions,
	).Scan(&p.ID)

	if err != nil {
		logger.WithField("error", err).
			Debug("unable to create project")
	}
	return err
}

// CreateGitRemote stores the passed in GitRemote in Postgres.
func (st *Postgres) CreateGitRemote(r *GitRemote) error {
	logger := logger.WithFields(logrus.Fields{
		"url":    r.URL,
		"branch": r.Branch,
	})
	logger.Debug("saving remote to postgres")

	sqlinsert := `
	INSERT INTO git_remotes (url, branch, project_id)
	VALUES
		($1, $2, $3)
	`

	_, err := st.db.Exec(sqlinsert, r.URL, r.Branch, r.ProjectID)

	if err != nil {
		logger.WithField("error", err).
			Debug("unable to create project")
	}
	return err
}

// GetProject retrieves the Project with the given id from postgres. If
// it's not found for the user it returns ErrProjectNotFound.
func (st *Postgres) GetProject(user string, id int) (Project, error) {
	logger := logger.WithField("project_id", id)
	logger.Debug("getting project from postgres")

	sqlq := `
	SELECT proj.id, proj.name, proj.description, proj.permissions,
		u.email, u.name, g.name,
		gr.url, gr.branch
	FROM projects AS proj
	INNER JOIN git_remotes AS gr
	ON proj.id = gr.project_id
	INNER JOIN users AS u
	ON proj.user_email = u.email
	INNER JOIN groups AS g
	ON u.group_name = g.name
	WHERE (u.email = $2
		OR u.group_name = proj.group_name AND (proj.permissions & 128) != 0
		OR (proj.permissions & 16) != 0)
		AND proj.id = $1;
	`

	rows, err := st.db.Query(sqlq, id, user)
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
		err := rows.Scan(&p.ID, &p.Name, &desc, &p.Permissions,
			&p.User.Email, &p.User.Name, &p.Group.Name,
			&gr.URL, &gr.Branch)
		if err != nil {
			logger.WithError(err).Debug("unable to scan row")
			return p, err
		}

		p.User.Group.Name = p.Group.Name

		if desc.Valid {
			p.Description = desc.String
		}

		p.GitRemotes = append(p.GitRemotes, gr)
	}

	return p, nil
}

// GetProjects retrieves all Projects from Postgres.
func (st *Postgres) GetProjects(user string) ([]Project, error) {
	logger.Debug("fetching all projects from postgres")

	// The 128 and 16 below correspond to the bits for "group read"
	// and "public read" permissions.
	sqlq := `
	SELECT p.id, p.name, p.description, p.permissions, u.email, u.name, g.name
	FROM projects AS p
	INNER JOIN users AS u
	ON p.user_email = u.email
	INNER JOIN groups AS g
	ON u.group_name = g.name
	WHERE u.email = $1
		OR u.group_name = p.group_name AND (p.permissions & 128) != 0
		OR (p.permissions & 16) != 0;
	`

	rows, err := st.db.Query(sqlq, user)
	if err != nil {
		logger.WithField("error", err).Debug("unable to query database")
		return nil, err
	}

	ps := []Project{}
	for rows.Next() {
		p := Project{}
		var desc sql.NullString
		err := rows.Scan(&p.ID, &p.Name, &desc, &p.Permissions,
			&p.User.Email, &p.User.Name, &p.User.Group.Name)
		if err != nil {
			logger.WithField("error", err).Debug("unable to scan row")
			return ps, err
		}

		// This needs to be populated correctly.
		p.Group.Name = p.User.Group.Name

		if desc.Valid {
			p.Description = desc.String
		}

		ps = append(ps, p)
	}

	return ps, nil
}

// GetPipelines implements the RelayStore interface. It returns a list of all
// pipelines for the project with the given id.
func (st *Postgres) GetPipelines(user string, pid int) ([]Pipeline, error) {
	sqlq := `
	SELECT p.id, p.name, p.remote_url, p.remote_branch, p.success
	FROM pipelines AS p
	INNER JOIN projects AS proj
	ON p.project_id = proj.id
	INNER JOIN users AS u
	ON proj.user_email = u.email
	INNER JOIN groups AS g
	ON u.group_name = g.name
	WHERE (u.email = $2
		OR u.group_name = proj.group_name AND (proj.permissions & 128) != 0
		OR (proj.permissions & 16) != 0)
		AND p.project_id = $1;
	`

	logger := logger.WithFields(log.Fields{
		"project_id": pid,
		"query":      "get_pipelines",
	})

	rows, err := st.db.Query(sqlq, pid, user)
	if err != nil {
		logger.WithError(err).Debug("unable to query postgres for pipelines")
	}

	ps := []Pipeline{}
	for rows.Next() {
		p := Pipeline{
			ProjectID: pid,
		}

		err := rows.Scan(&p.ID, &p.Name, &p.GitRemote.URL, &p.GitRemote.Branch, &p.Success)
		if err != nil {
			logger.WithError(err).Debug("unable to scan row")

			return ps, err
		}

		ps = append(ps, p)
	}

	return ps, nil
}

// GetPipeline retrieves the Pipeline with the given id from postgres.
func (st *Postgres) GetPipeline(user string, id int) (Pipeline, error) {
	logger := logger.WithField("id", id)
	logger.Debug("getting pipeline from postgres")

	sqlq := `
	SELECT p.name, p.success, p.remote_url, p.remote_branch, p.project_id,
		r.count, r.start_time, r.end_time, r.success
	FROM pipelines AS p
	INNER JOIN runs AS r
	ON p.id = r.pipeline_id
	INNER JOIN projects AS proj
	ON p.project_id = proj.id
	INNER JOIN users AS u
	ON proj.user_email = u.email
	INNER JOIN groups AS g
	ON u.group_name = g.name
	WHERE (u.email = $2
		OR u.group_name = proj.group_name AND (proj.permissions & 128) != 0
		OR (proj.permissions & 16) != 0)
		AND p.id = $1;
	`

	var p Pipeline
	rows, err := st.db.Query(sqlq, id, user)
	if err != nil {
		logger.WithError(err).Debug("unable to query database")
		return p, err
	}

	for rows.Next() {
		r := Run{PipelineID: id}

		// It's safe to always overwrite `p` here because these values
		// should always be the same.
		err := rows.Scan(&p.Name, &p.Success, &p.GitRemote.URL, &p.GitRemote.Branch, &p.ProjectID,
			&r.Count, &r.Start, &r.End, &r.Success)
		if err != nil {
			logger.WithError(err).Debug("unable to scan row")
			return p, err
		}

		p.Runs = append(p.Runs, r)
	}

	p.ID = id

	return p, err
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

// GetRun returns the nth run of the pipeline with the given ID. If the run
// isn't found it returns ErrRunNotFound.
func (st *Postgres) GetRun(pid, n int) (Run, error) {
	logger := logger.WithFields(logrus.Fields{
		"pipeline_id": pid,
		"count":       n,
	})
	logger.Debug("getting run from postgres")

	sqlq := `
	SELECT r.start_time, r.end_time, r.success,
		s.id, s.name, s.start_time, s.end_time, s.success
	FROM runs AS r
	INNER JOIN steps AS s
	ON r.count = s.run_count
		AND r.pipeline_id = s.pipeline_id
	WHERE r.pipeline_id = $1 AND r.count = $2
	`

	r := Run{
		PipelineID: pid,
		Count:      n,
	}
	rows, err := st.db.Query(sqlq, pid, n)
	if err != nil {
		// TODO: if this is ErrNoRows return ErrRunNotFound.
		logger.WithError(err).Debug("unable to query database")
		return r, err
	}

	for rows.Next() {
		s := Step{
			PipelineID: pid,
			RunCount:   n,
		}

		// It's safe to always overwrite `r` here because these values
		// should always be the same.
		err := rows.Scan(&r.Start, &r.End, &r.Success,
			&s.ID, &s.Name, &s.Start, &s.End, &s.Success)
		if err != nil {
			logger.WithError(err).Debug("unable to scan row")
			return r, err
		}

		r.Steps = append(r.Steps, s)
	}

	return r, nil
}

// GetStep returns the nth run of the pipeline with the given ID. If the Step
// isn't found it returns ErrStepNotFound.
func (st *Postgres) GetStep(id int) (Step, error) {
	logger := logger.WithField("id", id)
	logger.Debug("getting step from postgres")

	sqlq := `
	SELECT s.name, s.start_time, s.end_time, s.success,
		t.id, t.name, t.start_time, t.end_time, t.success
	FROM steps AS s
	INNER JOIN tasks AS t
	ON s.id = t.step_id
	WHERE s.id = $1
	`

	s := Step{ID: id}
	rows, err := st.db.Query(sqlq, id)
	if err != nil {
		logger.WithError(err).Debug("unable to query database")
		return s, err
	}

	if !rows.Next() {
		return s, ErrStepNotFound
	}

	// The loop has to be unrolled to handle the first call to
	// Next() that was used to check for a result.
	t := Task{StepID: id}
	err = rows.Scan(&s.Name, &s.Start, &s.End, &s.Success,
		&t.ID, &t.Name, &t.Start, &t.End, &t.Success)
	if err != nil {
		logger.WithError(err).Debug("unable to scan row")
		return s, err
	}

	s.Tasks = append(s.Tasks, t)

	for rows.Next() {
		t := Task{StepID: id}

		// It's safe to always overwrite `s` here because these values
		// should always be the same.
		err := rows.Scan(&s.Name, &s.Start, &s.End, &s.Success,
			&t.ID, &t.Name, &t.Start, &t.End, &t.Success)
		if err != nil {
			logger.WithError(err).Debug("unable to scan row")
			return s, err
		}

		s.Tasks = append(s.Tasks, t)
	}

	return s, nil
}

// GetTask returns the Task with the given ID. If the Task
// isn't found it returns ErrTaskNotFound.
func (st *Postgres) GetTask(id int) (Task, error) {
	logger := logger.WithField("id", id)
	logger.Debug("getting Task from postgres")

	sqlq := `
	SELECT name, start_time, end_time, success, step_id
	FROM tasks
	WHERE tasks.id = $1
	`

	t := Task{ID: id}
	err := st.db.QueryRow(sqlq, id).Scan(&t.Name, &t.Start, &t.End, &t.Success, &t.StepID)
	if err != nil {
		logger.WithError(err).Debug("unable to query row")
		if err == sql.ErrNoRows {
			return t, ErrTaskNotFound
		}
	}

	return t, err
}

// CreateGroup creates the passed in group in the database.
func (st *Postgres) CreateGroup(g *Group) error {
	logger := logger.WithField("name", g.Name)
	logger.Debug("saving group")

	sqlq := `
	INSERT INTO groups (name)
	VALUES
		($1)
	`

	_, err := st.db.Exec(sqlq, g.Name)
	return err
}

// CreateUser creates the passed in user in the database.
func (st *Postgres) CreateUser(u *User) error {
	logger := logger.WithField("email", u.Email)
	logger.Debug("saving user")

	if u.Group.Name == "" {
		logger.Debugf("got user with no group, setting to %v", DefaultGroup.Name)
		u.Group = DefaultGroup
	}

	password, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.WithError(err).Debug("unable to encrypt password")
		return err
	}

	sqlq := `
	INSERT INTO users (email, name, password, group_name)
	VALUES
		($1, $2, $3, $4)
	`

	_, err = st.db.Exec(sqlq, u.Email, u.Name, password, u.Group.Name)
	return err
}

// Authenticate checks the password for the user with the given email address.
func (st *Postgres) Authenticate(email, pass string) error {
	logger := logger.WithField("email", email)
	logger.Debug("authenticating user")

	sqlq := `
	SELECT password
	FROM users
	WHERE users.email = $1
	`

	cryptpass := []byte{}
	err := st.db.QueryRow(sqlq, email).Scan(&cryptpass)
	if err != nil {
		logger.WithError(err).Debug("unable to query row")
		if err == sql.ErrNoRows {
			return ErrNotAuthenticated
		}
	}

	err = bcrypt.CompareHashAndPassword(cryptpass, []byte(pass))
	if err != nil {
		logger.WithError(err).Debug("unable to authenticate")
		return ErrNotAuthenticated
	}

	return nil
}
