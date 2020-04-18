/*
 * Copyright 2020 Umputun. All rights reserved.
 * Use of this source code is governed by a MIT-style
 * license that can be found in the LICENSE file.
 */

package server

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/go-pkgz/jrpc"
	"github.com/stretchr/testify/assert"

	"github.com/umputun/remark/backend/app/store/image"
)

// gopher png for test, from https://golang.org/src/image/png/example_test.go
const gopher = "iVBORw0KGgoAAAANSUhEUgAAAEsAAAA8CAAAAAALAhhPAAAFfUlEQVRYw62XeWwUVRzHf2" +
	"+OPbo9d7tsWyiyaZti6eWGAhISoIGKECEKCAiJJkYTiUgTMYSIosYYBBIUIxoSPIINEBDi2VhwkQrVsj1ESgu9doHWdrul7ba" +
	"73WNm3vOPtsseM9MdwvvrzTs+8/t95ze/33sI5BqiabU6m9En8oNjduLnAEDLUsQXFF8tQ5oxK3vmnNmDSMtrncks9Hhtt" +
	"/qeWZapHb1ha3UqYSWVl2ZmpWgaXMXGohQAvmeop3bjTRtv6SgaK/Pb9/bFzUrYslbFAmHPp+3WhAYdr+7GN/YnpN46Opv55VDs" +
	"JkoEpMrY/vO2BIYQ6LLvm0ThY3MzDzzeSJeeWNyTkgnIE5ePKsvKlcg/0T9QMzXalwXMlj54z4c0rh/mzEfr+FgWEz2w6uk" +
	"8dkzFAgcARAgNp1ZYef8bH2AgvuStbc2/i6CiWGj98y2tw2l4FAXKkQBIf+exyRnteY83LfEwDQAYCoK+P6bxkZm/0966LxcAA" +
	"ILHB56kgD95PPxltuYcMtFTWw/FKkY/6Opf3GGd9ZF+Qp6mzJxzuRSractOmJrH1u8XTvWFHINNkLQLMR+XHXvfPPHw967raE1xxwtA36I" +
	"MRfkAAG29/7mLuQcb2WOnsJReZGfpiHsSBX81cvMKywYZHhX5hFPtOqPGWZCXnhWGAu6lX91ElKXSalcLXu3UaOXVay57ZSe5f6Gpx7J2" +
	"MXAsi7EqSp09b/MirKSyJfnfEEgeDjl8FgDAfvewP03zZ+AJ0m9aFRM8eEHBDRKjfcreDXnZdQuAxXpT2NRJ7xl3UkLBhuVGU16gZiGOgZm" +
	"rSbRdqkILuL/yYoSXHHkl9KXgqNu3PB8oRg0geC5vFmLjad6mUyTKLmF3OtraWDIfACyXqmephaDABawfpi6tqqBZytfQMqOz6S09iWXhkt" +
	"rRaB8Xz4Yi/8gyABDm5NVe6qq/3VzPrcjELWrebVuyY2T7ar4zQyybUCtsQ5Es1FGaZVrRVQwAgHGW2ZCRZshI5bGQi7HesyE972pOSeMM0" +
	"dSktlzxRdrlqb3Osa6CCS8IJoQQQgBAbTAa5l5epO34rJszibJI8rxLfGzcp1dRosutGeb2VDNgqYrwTiPNsLxXiPi3dz7LiS1WBRBDBOnqEj" +
	"yy3aQb+/bLiJzz9dIkscVBBLxMfSEac7kO4Fpkngi0ruNBeSOal+u8jgOuqPz12nryMLCniEjtOOOmpt+KEIqsEdocJjYXwrh9OZqWJQyPCTo67" +
	"LNS/TdxLAv6R5ZNK9npEjbYdT33gRo4o5oTqR34R+OmaSzDBWsAIPhuRcgyoteNi9gF0KzNYWVItPf2TLoXEg+7isNC7uJkgo1iQWOfRSP9NR" +
	"11RtbZZ3OMG/VhL6jvx+J1m87+RCfJChAtEBQkSBX2PnSiihc/Twh3j0h7qdYQAoRVsRGmq7HU2QRbaxVGa1D6nIOqaIWRjyRZpHMQKWKpZM5fe" +
	"A+lzC4ZFultV8S6T0mzQGhQohi5I8iw+CsqBSxhFMuwyLgSwbghGb0AiIKkSDmGZVmJSiKihsiyOAUs70UkywooYP0bii9GdH4sfr1UNysd3fU" +
	"yLLMQN+rsmo3grHl9VNJHbbwxoa47Vw5gupIqrZcjPh9R4Nye3nRDk199V+aetmvVtDRE8/+cbgAAgMIWGb3UA0MGLE9SCbWX670TDy" +
	"1y98c3D27eppUjsZ6fql3jcd5rUe7+ZIlLNQny3Rd+E5Tct3WVhTM5RBCEdiEK0b6B+/ca2gYU393nFj/n1AygRQxPIUA043M42u85+z2S" +
	"nssKrPl8Mx76NL3E6eXc3be7OD+H4WHbJkKI8AU8irbITQjZ+0hQcPEgId/Fn/pl9crKH02+5o2b9T/eMx7pKoskYgAAAABJRU5ErkJggg=="

func gopherPNG() io.Reader { return base64.NewDecoder(base64.StdEncoding, strings.NewReader(gopher)) }
func gopherPNGBytes() []byte {
	img, _ := ioutil.ReadAll(gopherPNG())
	return img
}

func TestRPC_imgLoadHndl(t *testing.T) {
	_, port, teardown := prepTestStore(t)
	defer teardown()
	api := fmt.Sprintf("http://localhost:%d/test", port)

	ri := image.RPC{Client: jrpc.Client{API: api, Client: http.Client{Timeout: 1 * time.Second}}}
	// save
	id := "test_img"
	err := ri.SaveWithID(id, gopherPNGBytes())
	assert.NoError(t, err)

	// load
	img, err := ri.Load(id)
	assert.NoError(t, err)
	assert.Equal(t, 1462, len(img))
	assert.Equal(t, gopherPNGBytes(), img)

	// commit
	err = ri.Commit(id)
	assert.NoError(t, err)

	// load after commit
	img, err = ri.Load(id)
	assert.NoError(t, err)
	assert.Equal(t, 1462, len(img))
	assert.Equal(t, gopherPNGBytes(), img)

	// cleanup
	err = ri.Cleanup(nil, time.Second)
	assert.NoError(t, err)

	// load after cleanup
	img, err = ri.Load(id)
	assert.NoError(t, err)
	assert.Equal(t, 1462, len(img))
	assert.Equal(t, gopherPNGBytes(), img)
}

func TestRPC_imgCommitHndlFail(t *testing.T) {
	_, port, teardown := prepTestStore(t)
	defer teardown()
	api := fmt.Sprintf("http://localhost:%d/test", port)

	ri := image.RPC{Client: jrpc.Client{API: api, Client: http.Client{Timeout: 1 * time.Second}}}
	err := ri.Commit("test_id")
	assert.EqualError(t, err, "failed to commit test_id, not found in staging")
}

func TestRPC_imgCleanupHndl(t *testing.T) {
	_, port, teardown := prepTestStore(t)
	defer teardown()
	api := fmt.Sprintf("http://localhost:%d/test", port)

	ri := image.RPC{Client: jrpc.Client{API: api, Client: http.Client{Timeout: 1 * time.Second}}}

	// save
	id := "test_img"
	err := ri.SaveWithID(id, gopherPNGBytes())
	assert.NoError(t, err)

	// load
	img, err := ri.Load(id)
	assert.NoError(t, err)
	assert.Equal(t, 1462, len(img))
	assert.Equal(t, gopherPNGBytes(), img)

	// cleanup
	err = ri.Cleanup(context.TODO(), time.Nanosecond)
	assert.NoError(t, err)

	// load after cleanup should fail
	_, err = ri.Load(id)
	assert.EqualError(t, err, "image test_img not found")
}
