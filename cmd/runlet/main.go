package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	nats "github.com/nats-io/go-nats"
	tasklog "github.com/run-ci/relay/cmd/runlet/log"
	"github.com/run-ci/relay/store"
	"github.com/run-ci/run/pkg/run"
	"github.com/run-ci/runlog"
	log "github.com/sirupsen/logrus"
)

var natsURL, gitimg, cimnt, pgconnstr string
var logger *log.Entry

func init() {
	logger = initlog()

	natsURL = os.Getenv("RELAY_NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}

	gitimg = os.Getenv("RELAY_GIT_IMAGE")
	if gitimg == "" {
		gitimg = "runci/git-clone"
	}

	cimnt = os.Getenv("RELAY_CI_MOUNT")
	if cimnt == "" {
		cimnt = "/ci/repo"
	}

	pgconnstr = initpg()
}

func main() {
	logger.Info("booting runlet")

	evq, teardown := SubscribeToQueue(natsURL, "pipelines", "runlet")
	defer teardown()

	st, err := store.NewPostgres(pgconnstr)
	if err != nil {
		logger.WithField("error", err).Fatal("unable to connect to postgres")
	}

	logger.Info("connecting to database")

	client, err := docker.NewClient("unix:///var/run/docker.sock")
	if err != nil {
		logger.WithFields(log.Fields{
			"error": err,
		}).Fatal("error opening docker socket")
	}

	logger.Info("initialized docker client")

	agent, err := run.NewAgent(client)
	if err != nil {
		logger.WithField("error", err).Fatal("unable to initialize run agent")
	}

	logger.Info("initialized run agent")

	for msg := range evq {
		logger.Debugf("processing message %s", msg.Data)

		var ev Event
		err := json.Unmarshal(msg.Data, &ev)
		if err != nil {
			logger.WithFields(log.Fields{
				"error": err,
			}).Error("error parsing message, skipping")

			continue
		}

		vol := initCIVolume(agent, client, ev.GitRemote)

		logger := logger.WithFields(log.Fields{
			"git_remote":    ev.GitRemote.URL,
			"git_branch":    ev.GitRemote.Branch,
			"pipeline_name": ev.Name,
		})

		logger.Debug("loading pipeline")

		var pipeline store.Pipeline
		pipeline.ID, err = st.GetPipelineID(ev.GitRemote, ev.Name)
		if err != nil && err != store.ErrNoPipelines {
			logger.WithField("error", err).Error("error loading pipeline from store, skipping this run")

			continue
		}
		if err == store.ErrNoPipelines {
			logger.Info("no pipeline found, creating one")

			p := store.Pipeline{
				Name:      ev.Name,
				GitRemote: ev.GitRemote,
			}
			err := st.CreatePipeline(&p)
			if err != nil {
				logger.WithField("error", err).Error("unable to create pipeline, skipping this run")

				continue
			}

			pipeline = p
		}

		logger.Debugf("got pipeline %+v", pipeline)

		r := store.Run{
			PipelineID: pipeline.ID,
		}
		r.SetStart()

		logger.Debug("creating new pipeline run")

		err = st.CreateRun(&r)
		if err != nil {
			logger.WithField("error", err).Error("unable to save pipeline run")

			continue
		}

		for _, step := range ev.Steps {
			// The pipeline could have been marked unsuccessful in some task. At
			// that point, the right thing to do is to break out of this loop.
			// Since the tasks are run in their own loop, they can't break to the
			// right spot, so this check needs to be here.
			if r.Failed() {
				break
			}

			logger := logger.WithFields(log.Fields{
				"step": step.Name,
			})

			logger.Debug("running step")

			start := time.Now()
			s := store.Step{
				Name:       step.Name,
				Start:      &start,
				RunCount:   r.Count,
				PipelineID: pipeline.ID,
			}
			r.Steps = append(r.Steps, s)

			err = st.CreateStep(&s)
			if err != nil {
				logger.WithField("error", err).Error("unable to save step, aborting")

				s.MarkSuccess(false)
				pipeline.MarkSuccess(false)

				break
			}

			for _, task := range step.Tasks {
				logger := logger.WithField("task", task.Name)

				logger.Debug("running task")

				start := time.Now()
				t := store.Task{
					Name:   task.Name,
					Start:  &start,
					StepID: s.ID,
				}
				s.Tasks = append(s.Tasks, t)

				err := st.CreateTask(&t)
				if err != nil {
					logger.WithField("error", err).Error("unable to save task, aborting")

					s.MarkSuccess(false)
					r.MarkSuccess(false)
					pipeline.MarkSuccess(false)

					break
				}

				if task.Mount == "" {
					task.Mount = cimnt

					logger.Debugf("mount point set to %v", task.Mount)
				}

				if task.Shell == "" {
					task.Shell = "sh"

					logger.Debugf("shell set to %v", task.Shell)
				}

				logger.Debug("initializing output logging chain")

				logoutPath := "logout.log"
				logout, err := os.OpenFile(logoutPath, os.O_CREATE|os.O_RDWR, 0644)
				if err != nil {
					logger.WithField("error", err).
						Errorf("unable to create file log at %v, won't log there", logoutPath)
				}

				logger.Debug("initializing error logging chain")

				logerrPath := "logerr.log"
				logerr, err := os.OpenFile(logerrPath, os.O_CREATE|os.O_RDWR, 0644)
				if err != nil {
					logger.WithField("error", err).
						Errorf("unable to create file log at %v, won't log there", logerrPath)
				}

				runlogClient := &runlog.Client{
					URL:      "runlog:9999",
					CertPath: "/tmp/devcerts/runlog.crt",
					KeyPath:  "/tmp/devcerts/runlog.key",
					CAPath:   "/tmp/devcerts/rootCA.pem",
					TaskID:   uint32(t.ID),
				}

				err = runlogClient.Connect()
				if err != nil {
					logger.WithError(err).Errorf("unable to connect to runlog: %v", err)
				}

				outchain := tasklog.Middleware(os.Stdout.Write).
					Chain(logout).
					Chain(runlogClient)
				errchain := tasklog.Middleware(os.Stdout.Write).
					Chain(logerr).
					Chain(runlogClient)

				spec := run.ContainerSpec{
					Imgref: task.Image,
					Cmd:    task.GetCmd(),
					Mount: run.Mount{
						Src:   vol,
						Point: task.Mount,
						Type:  "volume",
					},

					OutputStream: outchain,
					ErrorStream:  errchain,
				}

				logger.Debug("running task container")

				id, status, err := agent.RunContainer(spec)
				logger = logger.WithField("container_id", id)
				if err != nil {
					logger.WithField("error", err).
						Error("error running task container")
				}

				logger.Debugf("task container exited with status %v", status)

				t.SetEnd()
				t.MarkSuccess(true)
				err = st.UpdateTask(&t)
				if err != nil {
					logger.WithField("error", err).Error("unable to save pipeline task, continuing")

					// Continuing here is safe because the task itself finished successfully.
				}
			}

			s.SetEnd()
			s.MarkSuccess(true)
			err = st.UpdateStep(&s)
			if err != nil {
				logger.WithField("error", err).Error("unable to save pipeline step, continuing")

				// Continuing here is safe because the step itself finished successfully.
			}
		}

		err = client.RemoveVolume(vol)
		if err != nil {
			logger.WithFields(log.Fields{
				"error": err,
				"vol":   vol,
			}).Error("unable to delete volume")
		}

		r.SetEnd()
		r.MarkSuccess(true)
		err = st.UpdateRun(&r)
		if err != nil {
			logger.WithFields(log.Fields{
				"error": err,
			}).Error("unable to save run")
		}

		pipeline.MarkSuccess(true)
		err = st.UpdatePipeline(&pipeline)
		if err != nil {
			logger.WithError(err).Error("unable to save pipeline")
		}
	}
}

func initpg() string {
	pguser := os.Getenv("RELAY_POSTGRES_USER")
	if pguser == "" {
		logger.Fatal("need RELAY_POSTGRES_USER")
	}

	pgpass := os.Getenv("RELAY_POSTGRES_PASS")
	if pgpass == "" {
		logger.Fatal("need RELAY_POSTGRES_PASS")
	}

	pghref := os.Getenv("RELAY_POSTGRES_HREF")
	if pghref == "" {
		logger.Fatal("need RELAY_POSTGRES_HREF")
	}

	pgdb := os.Getenv("RELAY_POSTGRES_DB")
	if pgdb == "" {
		logger.Fatal("need RELAY_POSTGRES_DB")
	}

	pgssl := os.Getenv("RELAY_POSTGRES_SSL")
	if pgssl == "" {
		logger.Info("RELAY_POSTGRES_SSL not set - defaulting to verify-full")
		pgssl = "verify-full"
	}

	return fmt.Sprintf("postgres://%v:%v@%v/%v?sslmode=%v",
		pguser, pgpass, pghref, pgdb, pgssl)
}

func initlog() *log.Entry {
	switch strings.ToLower(os.Getenv("RELAY_LOG_LEVEL")) {
	case "debug", "trace":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn", "warning":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "fatal":
		log.SetLevel(log.FatalLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}

	return log.WithFields(log.Fields{
		"package": "main",
	})
}
