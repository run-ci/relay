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

	connstr := args[0]
	if connstr == "" {
		usage()
		return
	}

	path := args[1]
	if path == "" {
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

	fmt.Printf("got data %+v", d)
}

type data struct {
	Groups   []store.Group
	Projects []store.Project
}
