package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/run-ci/relay/store"
	yaml "gopkg.in/yaml.v2"
)

func usage() {
	fmt.Println("usage: go run dev/seed-db/main.go $POSTGRES_CONNECTION_STRING $DATA_YAML_PATH")
}

func main() {
	// This is 4 because passing arguments to `go run` requires the `--` and
	// that also counts as one of the arguments in `os.Args`.
	if len(os.Args) != 4 {
		usage()
		os.Exit(1)
	}

	args := os.Args[2:]

	path := args[0]
	if path == "" {
		usage()
		return
	}

	connstr := args[1]
	if connstr == "" {
		usage()
		return
	}

	fmt.Printf("seeding %v with data from %v\n", connstr, path)

	f, err := os.Open(path)
	if err != nil {
		fmt.Printf("got error reading path: %v\n", err)
		os.Exit(1)
	}

	buf, err := ioutil.ReadAll(f)
	if err != nil {
		fmt.Printf("got error reading file: %v\n", err)
		os.Exit(1)
	}

	var d data
	err = yaml.Unmarshal(buf, &d)
	if err != nil {
		fmt.Printf("got error loading YAML: %v", err)
		os.Exit(1)
	}

	db, err := store.NewPostgres(connstr)
	if err != nil {
		fmt.Printf("got error connecting to postgres: %v", err)
		os.Exit(1)
	}

	fmt.Println("inserting default group")
	err = db.CreateGroup(&store.DefaultGroup)
	if err != nil {
		fmt.Printf("got error inserting group: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("inserting default user")
	err = db.CreateUser(&store.DefaultUser)
	if err != nil {
		fmt.Printf("got error inserting default user: %v\n", err)
		os.Exit(1)
	}

	for _, proj := range d.Projects {
		fmt.Printf("inserting project %v\n", proj.Name)

		err := db.CreateProject(&proj)
		if err != nil {
			fmt.Printf("got error inserting project: %v\n", err)
			os.Exit(1)
		}

		for _, remote := range proj.GitRemotes {
			fmt.Printf("inserting git remote %v#%v\n", remote.URL, remote.Branch)

			remote.ProjectID = proj.ID
			err := db.CreateGitRemote(&remote)
			if err != nil {
				fmt.Printf("error inserting git remote: %v\n", err)
				os.Exit(1)
			}
		}
	}

	fmt.Println("done!")
}

type data struct {
	Groups   []store.Group
	Projects []store.Project
}
