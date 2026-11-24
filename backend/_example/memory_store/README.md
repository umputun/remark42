# sample store implementation

`memory_store` illustrates how to make a custom storage plugin for remark42.

In order to run remark42 with memory_store, run from the repository root:

1. `docker compose -f backend/_example/memory_store/compose-dev-memstore.yml build`
1. `docker compose -f backend/_example/memory_store/compose-dev-memstore.yml up`

As usual, demo site will run on http://127.0.0.1:8080/web/

note: in order to work with the latest (current) version of master `go.mod` uses replacement directive for the backend package. In real-life usage `replace github.com/umputun/remark42/backend => ../../`  should not be used. 
