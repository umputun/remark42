package image

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const proxyAPI = "https://remark42.example/api/v1/img"

func proxiedSrc(imgURL string) string {
	return proxyAPI + "?src=" + base64.URLEncoding.EncodeToString([]byte(imgURL))
}

func external(t *testing.T, imgURL string) External {
	t.Helper()
	id, err := CachedImgID(imgURL)
	require.NoError(t, err)
	return External{ID: id, URL: imgURL}
}

// memStore keeps staged and committed images in memory
type memStore struct {
	mu                sync.Mutex
	staged, committed map[string][]byte
	resets            map[string]int // ResetCleanupTimer calls per id
}

func newMemStore() *memStore {
	return &memStore{staged: map[string][]byte{}, committed: map[string][]byte{}, resets: map[string]int{}}
}

func (m *memStore) Info() (StoreInfo, error) { return StoreInfo{}, nil }
func (m *memStore) Save(id string, img []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.staged[id] = img
	return nil
}

func (m *memStore) Load(id string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if img, ok := m.committed[id]; ok {
		return img, nil
	}
	if img, ok := m.staged[id]; ok {
		return img, nil
	}
	return nil, fmt.Errorf("no image %s", id)
}

func (m *memStore) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.staged, id)
	delete(m.committed, id)
	return nil
}
func (m *memStore) ResetCleanupTimer(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.resets[id]++
	return nil
}

func (m *memStore) resetCount(id string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.resets[id]
}
func (m *memStore) Commit(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	img, ok := m.staged[id]
	if !ok {
		if _, done := m.committed[id]; done {
			return nil
		}
		return fmt.Errorf("no staged image %s", id)
	}
	delete(m.staged, id)
	m.committed[id] = img
	return nil
}
func (m *memStore) Cleanup(context.Context, time.Duration) error { return nil }

func (m *memStore) state(id string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.committed[id]; ok {
		return "committed"
	}
	if _, ok := m.staged[id]; ok {
		return "staged"
	}
	return "none"
}

// fakeFetcher answers every fetch with the gopher image and records the urls; a url in block waits for
// release first
type fakeFetcher struct {
	mu      sync.Mutex
	fetched []string
	block   map[string]bool
	release chan struct{}
}

func (f *fakeFetcher) fetch(ctx context.Context, imgURL string) ([]byte, error) {
	f.mu.Lock()
	f.fetched = append(f.fetched, imgURL)
	blocked := f.block[imgURL]
	f.mu.Unlock()
	if blocked {
		select {
		case <-f.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return gopherPNGBytes(), nil
}

func (f *fakeFetcher) urls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.fetched...)
}

func TestService_ExternalPictures(t *testing.T) {
	svc := NewService(newMemStore(), ServiceParams{ProxyAPI: proxyAPI, ImageAPI: "https://remark42.example/api/v1/picture/"})
	html := `<img src="` + proxiedSrc("https://a.example/1.png") + `">` +
		`<img src="https://remark42.example/api/v1/picture/dev/abc">` +
		`<img src="https://b.example/plain.png">` +
		`<img src="` + proxiedSrc("https://a.example/1.png") + `">` +
		`<img src="` + proxyAPI + `?src=not-base64!">` +
		`<img src="` + proxiedSrc("https://a.example/2.png") + `">`
	assert.Equal(t, []External{external(t, "https://a.example/1.png"), external(t, "https://a.example/2.png")},
		svc.ExternalPictures(html), "proxied images only, once each")

	html = ""
	for i := range maxExternalPerComment + 5 {
		html += `<img src="` + proxiedSrc(fmt.Sprintf("https://a.example/%d.png", i)) + `">`
	}
	assert.Len(t, svc.ExternalPictures(html), maxExternalPerComment, "capped per comment")

	assert.Nil(t, NewService(newMemStore(), ServiceParams{}).ExternalPictures(html), "no proxy, no external images")
	var nilSvc *Service
	assert.Nil(t, nilSvc.ExternalPictures(html))
}

func TestService_SubmitExternal(t *testing.T) {
	a, b := "https://a.example/a.png", "https://a.example/b.png"
	newSvc := func(editDuration time.Duration) (*Service, *memStore, *fakeFetcher) {
		store, fetcher := newMemStore(), &fakeFetcher{}
		svc := NewService(store, ServiceParams{ProxyAPI: proxyAPI, EditDuration: editDuration, MaxSize: 10000,
			FetchExternal: fetcher.fetch})
		return svc, store, fetcher
	}

	t.Run("staged before the comment's commit, which then keeps it", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			svc, store, fetcher := newSvc(time.Hour)
			ids := func() []string { return []string{external(t, a).ID} }
			svc.Submit(ids)
			svc.SubmitExternal("u1", time.Now(), []External{external(t, a)}, ids)
			synctest.Wait()
			assert.Equal(t, []string{a}, fetcher.urls())
			assert.Equal(t, "staged", store.state(external(t, a).ID), "left for the delayed commit")
			time.Sleep(time.Hour)
			synctest.Wait()
			assert.Equal(t, "committed", store.state(external(t, a).ID))
			svc.Close(context.Background())
		})
	})

	t.Run("fetched after the commit ran, committed if the comment still shows it", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			svc, store, _ := newSvc(0)
			shown := []string{external(t, a).ID}
			svc.SubmitExternal("u1", time.Now(), []External{external(t, a), external(t, b)}, func() []string { return shown })
			synctest.Wait()
			assert.Equal(t, "committed", store.state(external(t, a).ID))
			assert.Equal(t, "staged", store.state(external(t, b).ID), "the comment no longer shows it")
			svc.Close(context.Background())
		})
	})

	t.Run("an image already stored is not fetched again, and its cleanup timer is reset", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			svc, store, fetcher := newSvc(time.Hour)
			require.NoError(t, store.Save(external(t, a).ID, []byte("stored")))
			svc.SubmitExternal("u1", time.Now(), []External{external(t, a), external(t, a)}, func() []string { return nil })
			synctest.Wait()
			assert.Empty(t, fetcher.urls())
			assert.Equal(t, 2, store.resetCount(external(t, a).ID), "a staged copy may be older than the comment")
			svc.Close(context.Background())
		})
	})

	t.Run("an image staged for another comment is committed for this one after its commit", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			svc, store, fetcher := newSvc(0)
			require.NoError(t, store.Save(external(t, a).ID, gopherPNGBytes()))
			svc.SubmitExternal("u1", time.Now(), []External{external(t, a)}, func() []string { return []string{external(t, a).ID} })
			synctest.Wait()
			assert.Empty(t, fetcher.urls())
			assert.Equal(t, "committed", store.state(external(t, a).ID))
			svc.Close(context.Background())
		})
	})

	t.Run("an image fetched while Close commits the comment is committed", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			svc, store, _ := newSvc(time.Hour)
			release := make(chan struct{})
			svc.FetchExternal = func(context.Context, string) ([]byte, error) { // a fetch Close's cancel comes too late for
				<-release
				return gopherPNGBytes(), nil
			}
			ids := func() []string { return []string{external(t, a).ID} }
			svc.Submit(ids)
			svc.SubmitExternal("u1", time.Now(), []External{external(t, a)}, ids)
			synctest.Wait()
			go svc.Close(context.Background())
			synctest.Wait() // the submitter has committed the comment, before the image was there
			close(release)
			synctest.Wait()
			assert.Equal(t, "committed", store.state(external(t, a).ID))
		})
	})

	t.Run("one user can't fill the queue", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			svc, store, fetcher := newSvc(time.Hour)
			fetcher.block, fetcher.release = map[string]bool{}, make(chan struct{})
			imgs := make([]External, 0, maxPendingExternalPerUser+10)
			for i := range maxPendingExternalPerUser + 10 {
				u := fmt.Sprintf("https://flood.example/%d.png", i)
				fetcher.block[u] = true
				imgs = append(imgs, external(t, u))
			}
			for i := 0; i < len(imgs); i += maxExternalPerComment {
				svc.SubmitExternal("flood", time.Now(), imgs[i:min(i+maxExternalPerComment, len(imgs))], func() []string { return nil })
			}
			svc.SubmitExternal("u2", time.Now(), []External{external(t, a)}, func() []string { return nil })
			synctest.Wait()
			assert.Len(t, fetcher.urls(), externalWorkers, "both fetchers are held by the flood")

			close(fetcher.release)
			synctest.Wait()
			assert.Len(t, fetcher.urls(), maxPendingExternalPerUser+1, "the flood's images past its share are dropped")
			assert.Equal(t, "staged", store.state(external(t, a).ID), "the other user's image is stored")

			svc.SubmitExternal("flood", time.Now(), []External{external(t, b)}, func() []string { return nil })
			synctest.Wait()
			assert.Equal(t, "staged", store.state(external(t, b).ID), "the user's share is free again once fetched")
			svc.Close(context.Background())
		})
	})

	t.Run("a failed fetch stores nothing", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			store := newMemStore()
			svc := NewService(store, ServiceParams{ProxyAPI: proxyAPI, MaxSize: 10000,
				FetchExternal: func(context.Context, string) ([]byte, error) { return nil, errors.New("refused") }})
			svc.SubmitExternal("u1", time.Now(), []External{external(t, a)}, func() []string { return []string{external(t, a).ID} })
			synctest.Wait()
			assert.Equal(t, "none", store.state(external(t, a).ID))
			svc.Close(context.Background())
		})
	})

	t.Run("nothing is fetched without a fetcher or after Close", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			store := newMemStore()
			svc := NewService(store, ServiceParams{ProxyAPI: proxyAPI, MaxSize: 10000})
			svc.SubmitExternal("u1", time.Now(), []External{external(t, a)}, func() []string { return nil })

			closed, _, fetcher := newSvc(time.Hour)
			closed.Close(context.Background())
			closed.SubmitExternal("u1", time.Now(), []External{external(t, a)}, func() []string { return nil })
			synctest.Wait()
			assert.Empty(t, fetcher.urls())
			assert.Equal(t, "none", store.state(external(t, a).ID))
			var nilSvc *Service
			nilSvc.SubmitExternal("u1", time.Now(), []External{external(t, a)}, nil)
		})
	})

	t.Run("Close stops the fetches in flight", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			svc, store, fetcher := newSvc(time.Hour)
			fetcher.block, fetcher.release = map[string]bool{a: true}, make(chan struct{})
			svc.SubmitExternal("u1", time.Now(), []External{external(t, a)}, func() []string { return nil })
			synctest.Wait()
			svc.Close(context.Background())
			assert.Equal(t, "none", store.state(external(t, a).ID))
		})
	})
}
