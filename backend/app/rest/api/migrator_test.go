package api

import (
	"compress/gzip"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/coreos/bbolt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark/backend/app/migrator"
	"github.com/umputun/remark/backend/app/store/engine"
	"github.com/umputun/remark/backend/app/store/service"
)

func TestMigrator_Import(t *testing.T) {
	srv, ts := prepImportSrv(t)
	assert.NotNil(t, srv)
	defer cleanupImportSrv(srv, ts)

	r := strings.NewReader(`{"id":"2aa0478c-df1b-46b1-b561-03d507cf482c","pid":"","text":"<p>test test #1</p>","user":{"name":"developer one","id":"dev","picture":"/api/v1/avatar/remark.image","profile":"https://remark42.com","admin":true,"ip":"ae12fe3b5f129b5cc4cdd2b136b7b7947c4d2741"},"locator":{"site":"radio-t","url":"https://radio-t.com/blah1"},"score":0,"votes":{},"time":"2018-04-30T01:37:00.849053725-05:00"}
	{"id":"83fd97fd-ff64-48d1-9fb7-ca7769c77037","pid":"p1","text":"<p>test test #2</p>","user":{"name":"developer one","id":"dev","picture":"/api/v1/avatar/remark.image","profile":"https://remark42.com","admin":true,"ip":"ae12fe3b5f129b5cc4cdd2b136b7b7947c4d2741"},"locator":{"site":"radio-t","url":"https://radio-t.com/blah2"},"score":0,"votes":{},"time":"2018-04-30T01:37:00.861387771-05:00"}`)

	client := &http.Client{Timeout: 1 * time.Second}
	req, err := http.NewRequest("POST", ts.URL+"/api/v1/admin/import?site=radio-t&provider=native&secret=123456", r)
	assert.Nil(t, err)
	resp, err := client.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	b, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, `{"size":2,"status":"ok"}`+"\n", string(b))
}

func TestMigrator_ImportRejected(t *testing.T) {
	srv, ts := prepImportSrv(t)
	assert.NotNil(t, srv)
	defer cleanupImportSrv(srv, ts)

	r := strings.NewReader(`{"id":"2aa0478c-df1b-46b1-b561-03d507cf482c","pid":"","text":"<p>test test #1</p>","user":{"name":"developer one","id":"dev","picture":"/api/v1/avatar/remark.image","profile":"https://remark42.com","admin":true,"ip":"ae12fe3b5f129b5cc4cdd2b136b7b7947c4d2741"},"locator":{"site":"radio-t","url":"https://radio-t.com/blah1"},"score":0,"votes":{},"time":"2018-04-30T01:37:00.849053725-05:00"}
	{"id":"83fd97fd-ff64-48d1-9fb7-ca7769c77037","pid":"p1","text":"<p>test test #2</p>","user":{"name":"developer one","id":"dev","picture":"/api/v1/avatar/remark.image","profile":"https://remark42.com","admin":true,"ip":"ae12fe3b5f129b5cc4cdd2b136b7b7947c4d2741"},"locator":{"site":"radio-t","url":"https://radio-t.com/blah2"},"score":0,"votes":{},"time":"2018-04-30T01:37:00.861387771-05:00"}`)

	client := &http.Client{Timeout: 1 * time.Second}
	req, err := http.NewRequest("POST", ts.URL+"/api/v1/admin/import?site=radio-t&provider=native&secret=XYZ", r)
	assert.Nil(t, err)
	resp, err := client.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestMigrator_Export(t *testing.T) {
	srv, ts := prepImportSrv(t)
	assert.NotNil(t, srv)
	defer cleanupImportSrv(srv, ts)

	r := strings.NewReader(`{"id":"2aa0478c-df1b-46b1-b561-03d507cf482c","pid":"","text":"<p>test test #1</p>","user":{"name":"developer one","id":"dev","picture":"/api/v1/avatar/remark.image","profile":"https://remark42.com","admin":true,"ip":"ae12fe3b5f129b5cc4cdd2b136b7b7947c4d2741"},"locator":{"site":"radio-t","url":"https://radio-t.com/blah1"},"score":0,"votes":{},"time":"2018-04-30T01:37:00.849053725-05:00"}
	{"id":"83fd97fd-ff64-48d1-9fb7-ca7769c77037","pid":"p1","text":"<p>test test #2</p>","user":{"name":"developer one","id":"dev","picture":"/api/v1/avatar/remark.image","profile":"https://remark42.com","admin":true,"ip":"ae12fe3b5f129b5cc4cdd2b136b7b7947c4d2741"},"locator":{"site":"radio-t","url":"https://radio-t.com/blah2"},"score":0,"votes":{},"time":"2018-04-30T01:37:00.861387771-05:00"}`)

	client := &http.Client{Timeout: 1 * time.Second}
	req, err := http.NewRequest("POST", ts.URL+"/api/v1/admin/import?site=radio-t&provider=native&secret=123456", r)
	require.Nil(t, err)
	resp, err := client.Do(req)
	require.Nil(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	req, err = http.NewRequest("GET", ts.URL+"/api/v1/admin/export?site=radio-t&secret=123456", nil)
	require.Nil(t, err)
	resp, err = client.Do(req)
	require.Nil(t, err)
	require.Equal(t, 200, resp.StatusCode)
	require.Equal(t, "application/gzip", resp.Header.Get("Content-Type"))

	ungzReader, err := gzip.NewReader(resp.Body)
	assert.NoError(t, err)
	ungzBody, err := ioutil.ReadAll(ungzReader)
	assert.NoError(t, err)
	assert.Equal(t, 2, strings.Count(string(ungzBody), "\n"))
	assert.Equal(t, 2, strings.Count(string(ungzBody), "\"text\""))
	t.Logf("%s", string(ungzBody))

	req, err = http.NewRequest("GET", ts.URL+"/api/v1/admin/export?site=radio-t&secret=bad", nil)
	require.Nil(t, err)
	resp, err = client.Do(req)
	require.Nil(t, err)
	require.Equal(t, 403, resp.StatusCode)
}

func TestMigrator_Shutdown(t *testing.T) {
	srv := Migrator{}
	go func() {
		time.Sleep(100 * time.Millisecond)
		srv.Shutdown()
	}()
	st := time.Now()
	srv.Run(0)
	assert.True(t, time.Since(st).Seconds() < 1, "should take about 100ms")
}

func prepImportSrv(t *testing.T) (svc *Migrator, ts *httptest.Server) {
	b, err := engine.NewBoltDB(bolt.Options{}, engine.BoltSite{FileName: testDb, SiteID: "radio-t"})
	require.Nil(t, err)
	dataStore := &service.DataStore{Interface: b}
	svc = &Migrator{
		DisqusImporter: &migrator.Disqus{DataStore: dataStore},
		NativeImporter: &migrator.Remark{DataStore: dataStore},
		NativeExported: &migrator.Remark{DataStore: dataStore},
		Cache:          &mockCache{},
		SecretKey:      "123456",
	}

	routes := svc.routes()
	ts = httptest.NewServer(routes)
	return svc, ts
}

func cleanupImportSrv(srv *Migrator, ts *httptest.Server) {
	ts.Close()
	os.Remove(testDb)
}
