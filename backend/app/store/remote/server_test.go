package remote

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer(t *testing.T) {
	s := Server{CommandURL: "/v1/cmd"}

	type respData struct {
		Res1 string
		Res2 bool
	}

	s.Add("test", func(params *json.RawMessage) Response {
		args := []interface{}{}
		if err := json.Unmarshal(*params, &args); err != nil {
			return Response{Error: err.Error()}
		}
		t.Logf("%+v", args)

		assert.Equal(t, 4, len(args))
		assert.Equal(t, "blah", args[0].(string))
		assert.Equal(t, 42., args[1].(float64))
		assert.Equal(t, true, args[2].(bool))
		assert.Equal(t, "", args[3].(time.Time))

		r, err := s.EncodeResponse(respData{"res blah", true})
		assert.NoError(t, err)
		return r
	})

	go func() { s.Run(9091) }()
	time.Sleep(10 * time.Millisecond)

	// check with direct http call
	clientReq := Request{Method: "test", Params: []interface{}{"blah", 42, true, time.Date(2018, 6, 9, 16, 7, 25, 0, time.UTC)}}
	b := bytes.Buffer{}
	require.NoError(t, json.NewEncoder(&b).Encode(clientReq))
	resp, err := http.Post("http://127.0.0.1:9091/v1/cmd", "application/json", &b)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	data, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, `{"result":{"Res1":"res blah","Res2":true}}`+"\n", string(data))

	// check with client call
	c := Client{API: "http://127.0.0.1:9091/v1/cmd", Client: http.Client{}}
	r, err := c.Call("test", "blah", 42, true, time.Date(2018, 6, 9, 16, 7, 25, 0, time.UTC))
	assert.NoError(t, err)
	assert.Equal(t, "", r.Error)

	res := respData{}
	err = json.Unmarshal(*r.Result, &res)
	assert.Equal(t, respData{Res1: "res blah", Res2: true}, res)

	assert.NoError(t, s.Shutdown())
}
