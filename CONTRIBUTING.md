# Contributing Guide

You can find all the components laid out in `cmd`. Everything at
the top level is just packages that are common to the whole CI
system, like for the database.

## General Guidelines

**Always** source an environment from the `env` directory.
For local development `env/local` works fine.

By default, Go build will use the module cache to build
the binaries. Because Run task containers are ephemeral,
this means that the entire module cache needs to be downloaded
on every build. You can avoid this by running the following
before you get started:

```
run vendor
export GO_MOD_BUILD_MODE=vendor
```

For testing behavior with NATS, you'll need a NATS client.
The NATS ruby gem works fine. You can install it with:

```bash
gem install nats
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

# In another window, trigger a pipeline run
nats-pub pipelines "$(cat examples/pipeline.json)"
```
