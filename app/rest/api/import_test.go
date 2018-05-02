package api

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/coreos/bbolt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark/app/migrator"
	"github.com/umputun/remark/app/store"
)

func TestImport(t *testing.T) {
	srv, port := prepImportSrv(t)
	assert.NotNil(t, srv)
	defer cleanupImportSrv(srv)

	r := strings.NewReader(`{"id":"2aa0478c-df1b-46b1-b561-03d507cf482c","pid":"","text":"<p>test test #1</p>","user":{"name":"developer one","id":"dev","picture":"/api/v1/avatar/remark.image","profile":"https://remark42.com","admin":true,"ip":"ae12fe3b5f129b5cc4cdd2b136b7b7947c4d2741"},"locator":{"site":"radio-t","url":"https://radio-t.com/blah1"},"score":0,"votes":{},"time":"2018-04-30T01:37:00.849053725-05:00"}
	{"id":"83fd97fd-ff64-48d1-9fb7-ca7769c77037","pid":"p1","text":"<p>test test #2</p>","user":{"name":"developer one","id":"dev","picture":"/api/v1/avatar/remark.image","profile":"https://remark42.com","admin":true,"ip":"ae12fe3b5f129b5cc4cdd2b136b7b7947c4d2741"},"locator":{"site":"radio-t","url":"https://radio-t.com/blah2"},"score":0,"votes":{},"time":"2018-04-30T01:37:00.861387771-05:00"}`)

	resp, err := http.Post(fmt.Sprintf("http://dev:password@127.0.0.1:%d/api/v1/admin/import?site=radio-t&provider=native",
		port), "application/json", r)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func prepImportSrv(t *testing.T) (srv *Import, port int) {
	dataStore, err := store.NewBoltDB(bolt.Options{}, store.BoltSite{FileName: testDb, SiteID: "radio-t"})
	require.Nil(t, err)

	srv = &Import{
		DisqusImporter: &migrator.Disqus{CommentCreator: dataStore},
		NativeImporter: &migrator.Remark{CommentCreator: dataStore},
		Cache:          &mockCache{},
	}

	portSetCh := make(chan bool)

	go func() {
		port = rand.Intn(50000) + 1025
		portSetCh <- true
		srv.Run(port)
	}()

	<-portSetCh

	time.Sleep(100 * time.Millisecond)
	return srv, port
}

func cleanupImportSrv(srv *Import) {
	srv.httpServer.Close()
	srv.httpServer.Shutdown(context.Background())
	os.Remove(testDb)
}
