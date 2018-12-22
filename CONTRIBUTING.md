# Contributing Guide

You can find all the components laid out in `cmd`. Everything at
the top level is just packages that are common to the whole CI
system, like for the database.

## Prerequisites

The only two things you should need to work with this repository
are the Docker daemon and the Run CLI.

Unfortunately, because the Run CLI doesn't support local networking
(yet), it's not possible to test builds locally using it.

For testing behavior with NATS, you'll need a NATS client.
The NATS ruby gem works fine. You can install it with:

```bash
gem install nats
```

## General Guidelines

**Always** source an environment from the `env` directory.
For local development `env/local` works fine.

For the build tasks to work, there needs to be an image
present in your Docker environment called `relay-dev` that
contains all the dependencies in the Go module cache. To
build that image, run the `vendor` task:

```
run vendor
```

## api-server

This is the interface to the database.

### Building and running

```
source env/local
run build-api
docker-compose down && docker-compose up

# In another window
nats-sub pollers

# In yet another window
source env/local
go run dev/seed-db/main.go -- dev/seed_data.yaml postgres://$POSTGRES_USER:$POSTGRES_PASSWORD@localhost:5432/$POSTGRES_DB?sslmode=$RELAY_POSTGRES_SSL
curl -XPOST -d@./examples/git-repo.json http://localhost:9001/repos/git
curl -XGET http://localhost:9001/repos/git
```

## runlet

This is the CI task runner.

### Building and running

```
source env/local
run build-runlet
docker-compose down && docker-compose up

# In another window, seed the database and trigger a pipeline run.
source env/local
go run dev/seed-db/main.go -- dev/seed_data.yaml postgres://$POSTGRES_USER:$POSTGRES_PASSWORD@localhost:5432/$POSTGRES_DB?sslmode=$RELAY_POSTGRES_SSL
nats-pub pipelines "$(cat examples/pipeline.json)"
```
