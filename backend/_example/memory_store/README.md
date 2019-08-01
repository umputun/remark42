# sample store implementation 

`memory_store` illustrates how to make a custom storage plugin for remark42. 

In order to run remark42 with memory_store copy provided `compose-dev-memstore.yml` to the root directory and run:

1. docker-compose -f compose-dev-memstore.yml build
1. docker-compose -f compose-dev-memstore.yml up

As usual, demo site will run on http://127.0.0.1:8080/web/

note: in order to work with the latest (current) version of master `go.mod` uses replacement directive for the backend package
. In real-life usage `replace github.com/umputun/remark/backend => ../../`  should not be used. 