package rpc

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerPrimitiveTypes(t *testing.T) {
	s := Server{API: "/v1/cmd"}

	type respData struct {
		Res1 string
		Res2 bool
	}

	s.Add("test", func(id uint64, params json.RawMessage) Response {
		args := []interface{}{}
		if err := json.Unmarshal(params, &args); err != nil {
			return Response{Error: err.Error()}
		}
		t.Logf("%+v", args)

		assert.Equal(t, 3, len(args))
		assert.Equal(t, "blah", args[0].(string))
		assert.Equal(t, 42., args[1].(float64))
		assert.Equal(t, true, args[2].(bool))

		r, err := s.EncodeResponse(id, respData{"res blah", true}, nil)
		assert.NoError(t, err)
		return r
	})

	go func() { _ = s.Run(9091) }()
	defer func() { assert.NoError(t, s.Shutdown()) }()
	time.Sleep(10 * time.Millisecond)

	// check with direct http call
	clientReq := Request{Method: "test", Params: []interface{}{"blah", 42, true}, ID: 123}
	b := bytes.Buffer{}
	require.NoError(t, json.NewEncoder(&b).Encode(clientReq))
	resp, err := http.Post("http://127.0.0.1:9091/v1/cmd", "application/json", &b)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	data, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, `{"result":{"Res1":"res blah","Res2":true},"id":123}`+"\n", string(data))

	// check with client call
	c := Client{API: "http://127.0.0.1:9091/v1/cmd", Client: http.Client{}}
	r, err := c.Call("test", "blah", 42, true)
	assert.NoError(t, err)
	assert.Equal(t, "", r.Error)

	res := respData{}
	err = json.Unmarshal(*r.Result, &res)
	assert.NoError(t, err)
	assert.Equal(t, respData{Res1: "res blah", Res2: true}, res)
	assert.Equal(t, uint64(1), r.ID)
}

func TestServerWithObject(t *testing.T) {
	s := Server{API: "/v1/cmd"}

	type respData struct {
		Res1 string
		Res2 bool
	}

	type reqData struct {
		Time time.Time
		F1   string
		F2   time.Duration
	}

	s.Add("test", func(id uint64, params json.RawMessage) Response {
		arg := reqData{}
		if err := json.Unmarshal(params, &arg); err != nil {
			return Response{Error: err.Error()}
		}
		t.Logf("%+v", arg)

		r, err := s.EncodeResponse(id, respData{"res blah", true}, nil)
		assert.NoError(t, err)
		return r
	})

	go func() { _ = s.Run(9091) }()
	defer func() { assert.NoError(t, s.Shutdown()) }()
	time.Sleep(10 * time.Millisecond)

	c := Client{API: "http://127.0.0.1:9091/v1/cmd", Client: http.Client{}}
	r, err := c.Call("test", reqData{Time: time.Now(), F1: "sawert", F2: time.Minute})
	assert.NoError(t, err)
	assert.Equal(t, "", r.Error)

	res := respData{}
	err = json.Unmarshal(*r.Result, &res)
	assert.NoError(t, err)
	assert.Equal(t, respData{Res1: "res blah", Res2: true}, res)
}

func TestServerMethodNotImplemented(t *testing.T) {
	s := Server{}
	ts := httptest.NewServer(http.HandlerFunc(s.handler))
	defer ts.Close()
	s.Add("test", func(id uint64, params json.RawMessage) Response {
		return Response{}
	})

	r := Request{Method: "blah"}
	buf := bytes.Buffer{}
	assert.NoError(t, json.NewEncoder(&buf).Encode(r))
	resp, err := http.Post(ts.URL, "application/json", &buf)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotImplemented, resp.StatusCode)

	assert.EqualError(t, s.Shutdown(), "http server is not running")
}

func TestServerWithAuth(t *testing.T) {
	s := Server{API: "/v1/cmd", AuthUser: "user", AuthPasswd: "passwd"}

	s.Add("test", func(id uint64, params json.RawMessage) Response {
		args := []interface{}{}
		if err := json.Unmarshal(params, &args); err != nil {
			return Response{Error: err.Error()}
		}
		t.Logf("%+v", args)

		assert.Equal(t, 3, len(args))
		assert.Equal(t, "blah", args[0].(string))
		assert.Equal(t, 42., args[1].(float64))
		assert.Equal(t, true, args[2].(bool))

		r, err := s.EncodeResponse(id, "res blah", nil)
		assert.NoError(t, err)
		return r
	})

	go func() { _ = s.Run(9091) }()
	time.Sleep(10 * time.Millisecond)
	defer func() { assert.NoError(t, s.Shutdown()) }()

	c := Client{API: "http://127.0.0.1:9091/v1/cmd", Client: http.Client{}, AuthUser: "user", AuthPasswd: "passwd"}
	r, err := c.Call("test", "blah", 42, true)
	assert.NoError(t, err)
	assert.Equal(t, "", r.Error)
	val := ""
	err = json.Unmarshal(*r.Result, &val)
	assert.NoError(t, err)
	assert.Equal(t, "res blah", val)

	c = Client{API: "http://127.0.0.1:9091/v1/cmd", Client: http.Client{}}
	_, err = c.Call("test", "blah", 42, true)
	assert.EqualError(t, err, "bad status 401 Unauthorized for test")
}

func TestServerErrReturn(t *testing.T) {
	s := Server{API: "/v1/cmd", AuthUser: "user", AuthPasswd: "passwd"}

	s.Add("test", func(id uint64, params json.RawMessage) Response {
		args := []interface{}{}
		if err := json.Unmarshal(params, &args); err != nil {
			return Response{Error: err.Error()}
		}
		t.Logf("%+v", args)

		assert.Equal(t, 3, len(args))
		assert.Equal(t, "blah", args[0].(string))
		assert.Equal(t, 42., args[1].(float64))
		assert.Equal(t, true, args[2].(bool))

		r, err := s.EncodeResponse(id, "res blah", errors.New("some error"))
		assert.NoError(t, err)
		return r
	})

	go func() { _ = s.Run(9091) }()
	defer func() { assert.NoError(t, s.Shutdown()) }()
	time.Sleep(10 * time.Millisecond)

	c := Client{API: "http://127.0.0.1:9091/v1/cmd", Client: http.Client{}, AuthUser: "user", AuthPasswd: "passwd"}
	_, err := c.Call("test", "blah", 42, true)
	assert.EqualError(t, err, "some error")
}

func TestServerGroup(t *testing.T) {
	s := Server{API: "/v1/cmd"}
	s.Group("pre", HandlersGroup{
		"fn1": func(id uint64, params json.RawMessage) Response {
			return Response{}
		},
		"fn2": func(id uint64, params json.RawMessage) Response {
			return Response{}
		},
	})
	go func() { _ = s.Run(9091) }()
	defer func() { assert.NoError(t, s.Shutdown()) }()
	time.Sleep(10 * time.Millisecond)

	c := Client{API: "http://127.0.0.1:9091/v1/cmd", Client: http.Client{}}
	_, err := c.Call("fn1")
	assert.EqualError(t, err, "bad status 501 Not Implemented for fn1")

	_, err = c.Call("pre.fn1")
	assert.NoError(t, err)
	_, err = c.Call("pre.fn2")
	assert.NoError(t, err)
}

func TestServerAddLate(t *testing.T) {
	s := Server{API: "/v1/cmd"}
	s.Add("fn1", func(id uint64, params json.RawMessage) Response {
		return Response{}
	})
	go func() { _ = s.Run(9091) }()
	defer func() { assert.NoError(t, s.Shutdown()) }()
	time.Sleep(10 * time.Millisecond)

	// too late, ignored after run
	s.Add("fn2", func(id uint64, params json.RawMessage) Response {
		return Response{}
	})

	c := Client{API: "http://127.0.0.1:9091/v1/cmd", Client: http.Client{}}
	_, err := c.Call("fn1")
	assert.NoError(t, err)
	_, err = c.Call("fn2")
	assert.EqualError(t, err, "bad status 501 Not Implemented for fn2")
}

func TestServerNoHandlers(t *testing.T) {
	s := Server{API: "/v1/cmd", AuthUser: "user", AuthPasswd: "passwd"}
	assert.EqualError(t, s.Run(9091), "nothing mapped for dispatch, Add has to be called prior to Run")
}
