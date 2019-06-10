package remote

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

	go func() { s.Run(9091) }()
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
	assert.Equal(t, respData{Res1: "res blah", Res2: true}, res)
	assert.Equal(t, uint64(1), r.ID)
	assert.NoError(t, s.Shutdown())
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

	go func() { s.Run(9091) }()
	time.Sleep(10 * time.Millisecond)

	c := Client{API: "http://127.0.0.1:9091/v1/cmd", Client: http.Client{}}
	r, err := c.Call("test", reqData{Time: time.Now(), F1: "sawert", F2: time.Minute})
	assert.NoError(t, err)
	assert.Equal(t, "", r.Error)

	res := respData{}
	err = json.Unmarshal(*r.Result, &res)
	assert.Equal(t, respData{Res1: "res blah", Res2: true}, res)

	assert.NoError(t, s.Shutdown())
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

	go func() { s.Run(9091) }()
	time.Sleep(10 * time.Millisecond)

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
	assert.EqualError(t, err, "bad status 401 for test")

	assert.NoError(t, s.Shutdown())
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

	go func() { s.Run(9091) }()
	time.Sleep(10 * time.Millisecond)

	c := Client{API: "http://127.0.0.1:9091/v1/cmd", Client: http.Client{}, AuthUser: "user", AuthPasswd: "passwd"}
	_, err := c.Call("test", "blah", 42, true)
	assert.EqualError(t, err, "some error")

	assert.NoError(t, s.Shutdown())
}

func TestServerNoHandlers(t *testing.T) {
	s := Server{API: "/v1/cmd", AuthUser: "user", AuthPasswd: "passwd"}
	assert.EqualError(t, s.Run(9091), "nothing mapped for dispatch, Add has to be called prior to Run")
}
