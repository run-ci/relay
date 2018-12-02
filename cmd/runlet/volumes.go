package main

import (
	"fmt"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/run-ci/run/pkg/run"
)

func initCIVolume(agent *run.Agent, client *docker.Client, remote string) string {
	logger := logger.WithField("remote", remote)

	name := fmt.Sprintf("runlet.%v", uuid.New())

	err := agent.VerifyImagePresent(gitimg, true)
	if err != nil {
		logger.WithField("error", err).
			Fatalf("unable to verify git-clone image presence")
	}

	vol, err := client.CreateVolume(docker.CreateVolumeOptions{
		Name: name,
	})
	if err != nil {
		logger.WithField("error", err).
			Fatalf("unable to create test volume")
	}

	logger = logger.WithField("vol", vol.Name)
	logger.Debugf("created volume: %v", vol.Name)

	spec := run.ContainerSpec{
		Imgref: gitimg,
		Cmd:    []string{remote, "."},
		Mount: run.Mount{
			Src:   vol.Name,
			Point: cimnt,
			Type:  "volume",
		},
	}

	logger.Debug("populating volumes")

	id, status, err := agent.RunContainer(spec)
	if err != nil {
		log.Fatalf("error running git clone container %v: %v", id, err)
	}

	logger.Debugf("git clone container exited with status %v", status)

	return name
}
