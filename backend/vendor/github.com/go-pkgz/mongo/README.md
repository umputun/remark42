# Mongo [![Build Status](https://travis-ci.org/go-pkgz/mongo.svg?branch=master)](https://travis-ci.org/go-pkgz/mongo) [![Go Report Card](https://goreportcard.com/badge/github.com/go-pkgz/mongo)](https://goreportcard.com/report/github.com/go-pkgz/mongo) [![Coverage Status](https://coveralls.io/repos/github/go-pkgz/mongo/badge.svg?branch=master)](https://coveralls.io/github/go-pkgz/mongo?branch=master)

Provides helpers on top of [mgo](https://github.com/globalsign/mgo)

## Install and update

`go get -u github.com/go-pkgz/mongo`


## Usage

- `Server` represents mongo instance and provides session accessor. Application usually creates one server object and uses it for anything needed with this particular mongo host or replica set.

- `Connection` encapsulates session and provides auto-closable wrapper. Each requests runs inside one of With* function makes new mongo session and closes on completion.

- `BufferedWriter` implements buffered writer to mongo. Write method caching internally till it reached buffer size. Flush methods can be called manually at any time.


```golang
    m, err := NewServerWithURL("mongodb://127.0.0.1:27017/test?debug=true", 3*time.Second)
    if err != nil {
        panic("can't make mongo server")
    } 
    
    type testRecord struct {
    	Key1 string
    	Kay2 int
    }
    
    err = c.WithCollection(func(coll *mgo.Collection) error { // create session
        // insert 100 records
        for i := 0; i < 100; i++ {
            r := testRecord{
                Key1: fmt.Sprintf("key-%02d", i%5),
                Key2: i,
            }
            if e := coll.Insert(r); e != nil {
                return e
            }
        }
        return nil
    })
    
```

## Dependencies 

- [globalsign/mgo](https://github.com/globalsign/mgo) - mgo mongo driver
- [stretchr/testify/](https://github.com/stretchr/testify) - testing library (test-only dependency)

## Testing

`testing.go` helps to create test for real mongo (not mocks)

- `mongo.MakeTestConnection` creates `mongo.Connection` for url defined in env `MONGO_TEST`. If not defined `mongodb://mongo:27017` used. By default it will use random connection with prefix `test_` in `test` DB.
- `mongo.RemoveTestCollection` - drops collection used by `MakeTestConnection`
- `mongo.RemoveTestCollections` - drops user-defined collections from `test` DB

