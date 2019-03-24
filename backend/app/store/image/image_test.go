package image

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_ExtractPictures(t *testing.T) {
	svc := Service{ImageAPI: "/blah/"}
	html := `blah <img src="/blah/user1/pic1.png"/> foo 
<img src="/blah/user2/pic3.png"/> xyz <p>123</p> <img src="/pic3.png"/>`
	ids, err := svc.ExtractPictures(html)
	require.NoError(t, err)
	assert.Equal(t, 2, len(ids), "two images")
	assert.Equal(t, "user1/pic1.png", ids[0])
	assert.Equal(t, "user2/pic3.png", ids[1])
}

func TestService_Cleanup(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := NewMockStore(ctrl)
	store.EXPECT().Cleanup(gomock.Any(), gomock.Any()).Times(10)

	svc := Service{Store: store, TTL: 100 * time.Millisecond}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*549)
	defer cancel()
	svc.Cleanup(ctx)
}

func TestService_Submit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := NewMockStore(ctrl)

	store.EXPECT().Commit(gomock.Any()).Times(5) // all 5 should be committed
	svc := Service{Store: store, ImageAPI: "/blah/", TTL: time.Millisecond * 100}
	svc.Submit([]string{"id1", "id2", "id3"})
	svc.Submit([]string{"id4", "id5"})
	svc.Submit(nil)
	time.Sleep(time.Millisecond * 500)
}

func TestService_Close(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := NewMockStore(ctrl)

	store.EXPECT().Commit(gomock.Any()).Times(5) // all 5 should be committed
	svc := Service{Store: store, ImageAPI: "/blah/", TTL: time.Millisecond * 500}
	svc.Submit([]string{"id1", "id2", "id3"})
	svc.Submit([]string{"id4", "id5"})
	svc.Submit(nil)
	svc.Close()
}

func TestService_SubmitDelay(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer func() {
		ctrl.Finish()
	}()

	store := NewMockStore(ctrl)

	store.EXPECT().Commit(gomock.Any()).Times(3) // first batch should be committed
	svc := Service{Store: store, ImageAPI: "/blah/", TTL: time.Millisecond * 100}
	svc.Submit([]string{"id1", "id2", "id3"})
	time.Sleep(150 * time.Millisecond) // let first batch to pass TTL
	svc.Submit([]string{"id4", "id5"})
	svc.Submit(nil)
}
