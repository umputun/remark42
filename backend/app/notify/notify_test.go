package notify

import (
	"errors"
	"fmt"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/store"
)

func TestService_NoDestinations(t *testing.T) {
	s := NewService(nil, 0)
	assert.Equal(t, defaultQueueSize, cap(s.queue))
	assert.NotNil(t, s)
	s.Submit(Request{Comment: store.Comment{ID: "123"}})
	s.Submit(Request{Comment: store.Comment{ID: "123"}})
	s.Submit(Request{Comment: store.Comment{ID: "123"}})
	s.Close()
}

func TestService_WithDestinations(t *testing.T) {
	d1, d2 := &MockDest{id: 1}, &MockDest{id: 2}
	s := NewService(nil, 1, d1, d2)
	assert.NotNil(t, s)

	s.Submit(Request{Comment: store.Comment{ID: "100"}})
	time.Sleep(time.Millisecond * 110)
	s.Submit(Request{Comment: store.Comment{ID: "101"}})
	time.Sleep(time.Millisecond * 110)
	s.Submit(Request{Comment: store.Comment{ID: "102"}})
	time.Sleep(time.Millisecond * 110)
	s.Close()

	require.Equal(t, 3, len(d1.Get()), "got all comments to d1")
	require.Equal(t, 3, len(d2.Get()), "got all comments to d2")

	assert.Equal(t, "100", d1.Get()[0].Comment.ID)
	assert.Equal(t, "101", d1.Get()[1].Comment.ID)
	assert.Equal(t, "102", d1.Get()[2].Comment.ID)
}

func TestService_WithDrops(t *testing.T) {
	d1, d2 := &MockDest{id: 1}, &MockDest{id: 2}
	s := NewService(nil, 1, d1, d2)
	assert.NotNil(t, s)

	s.Submit(Request{Comment: store.Comment{ID: "100"}})
	s.Submit(Request{Comment: store.Comment{ID: "101"}})
	s.Submit(Request{Comment: store.Comment{ID: "102"}})
	time.Sleep(time.Millisecond * 21)
	s.Close()

	s.Submit(Request{Comment: store.Comment{ID: "111"}}) // safe to send after close

	assert.LessOrEqual(t, len(d1.Get()), 2, "at least one comment from three dropped from d1, got: %v", d1.Get())
	assert.LessOrEqual(t, len(d2.Get()), 2, "at least one comment from three dropped from d2, got: %v", d2.Get())
}

func TestService_SubmitVerificationWithDrops(t *testing.T) {
	d1, d2 := &MockDest{id: 1}, &MockDest{id: 2}
	s := NewService(nil, 1, d1, d2)
	assert.NotNil(t, s)

	s.SubmitVerification(VerificationRequest{
		SiteID: "remark",
		User:   "testUser",
		Email:  "test@example.org",
		Token:  "testToken",
	})
	s.SubmitVerification(VerificationRequest{})
	s.SubmitVerification(VerificationRequest{})
	time.Sleep(time.Millisecond * 21)
	s.Close()

	s.SubmitVerification(VerificationRequest{}) // safe to send after close

	assert.LessOrEqual(t, len(d2.GetVerify()), 2, "one request from three dropped from d2, got: %v", d2.GetVerify())

	verifyDest := d1.GetVerify()
	require.LessOrEqual(t, len(verifyDest), 2, "one request from three dropped from d1, got: %v", verifyDest)
	assert.Equal(t, "remark", verifyDest[0].SiteID)
	assert.Equal(t, "testUser", verifyDest[0].User)
	assert.Equal(t, "test@example.org", verifyDest[0].Email)
	assert.Equal(t, "testToken", verifyDest[0].Token)
}

func TestService_Many(t *testing.T) {
	d1, d2 := &MockDest{id: 1}, &MockDest{id: 2}
	s := NewService(nil, 5, d1, d2)
	assert.NotNil(t, s)

	for i := 0; i < 10; i++ {
		s.Submit(Request{Comment: store.Comment{ID: fmt.Sprintf("%d", 100+i)}})
		s.SubmitVerification(VerificationRequest{User: fmt.Sprintf("%d", 100+i)})
		time.Sleep(time.Millisecond * time.Duration(rand.Int31n(20)))
	}
	s.Close()
	time.Sleep(time.Millisecond * 10)

	assert.NotEqual(t, 10, len(d1.Get()), "some comments dropped from d1")
	assert.NotEqual(t, 10, len(d1.GetVerify()), "some verifications dropped from d1")
	assert.NotEqual(t, 10, len(d2.Get()), "some comments dropped from d2")
	assert.NotEqual(t, 10, len(d2.GetVerify()), "some verifications dropped from d2")

	assert.True(t, d1.closed)
	assert.True(t, d2.closed)
	assert.Equal(t, "mock id=1, closed=true", d1.String())
}

func TestService_WithParent(t *testing.T) {
	dest := &MockDest{id: 1}
	dataStore := &mockStore{data: map[string]store.Comment{}}

	dataStore.data["p1"] = store.Comment{ID: "p1"}
	dataStore.data["p2"] = store.Comment{ID: "p2"}

	s := NewService(dataStore, 1, dest)
	assert.NotNil(t, s)

	s.Submit(Request{Comment: store.Comment{ID: "c1", ParentID: "p1"}})
	time.Sleep(time.Millisecond * 110)
	s.Submit(Request{Comment: store.Comment{ID: "c11", ParentID: "p11"}})
	time.Sleep(time.Millisecond * 110)
	s.Close()

	destRes := dest.Get()
	require.Equal(t, 2, len(destRes), "two comment notified")
	assert.Equal(t, "p1", destRes[0].Comment.ParentID)
	assert.Equal(t, "p1", destRes[0].parent.ID)
	assert.Equal(t, "p11", destRes[1].Comment.ParentID)
	assert.Equal(t, "", destRes[1].parent.ID)
}

func TestService_EmailRetrieval(t *testing.T) {
	dest := &MockDest{id: 1}
	dataStore := &mockStore{data: map[string]store.Comment{}, emailData: map[string]string{}}

	dataStore.data["p1"] = store.Comment{ID: "p1", User: store.User{ID: "u1"}}
	dataStore.data["p2"] = store.Comment{ID: "p2", ParentID: "p1", User: store.User{ID: "u1"}}
	dataStore.data["p3"] = store.Comment{ID: "p3", ParentID: "p1", User: store.User{ID: "u2"}}
	dataStore.data["p4"] = store.Comment{ID: "p4", ParentID: "p3", User: store.User{ID: "u1"}}
	dataStore.emailData["u1"] = "u1@example.com"

	s := NewService(dataStore, 1, dest)
	assert.NotNil(t, s)

	// one comment, one notification
	s.Submit(Request{Comment: dataStore.data["p1"]})
	time.Sleep(time.Millisecond * 110)

	destRes := dest.Get()
	require.Equal(t, 1, len(destRes), "one comment notified")
	assert.Equal(t, "p1", destRes[0].Comment.ID)
	assert.Empty(t, destRes[0].parent)
	assert.Empty(t, destRes[0].Emails)

	// reply to the first comment, same comment as one in original comment
	s.Submit(Request{Comment: dataStore.data["p2"]})
	time.Sleep(time.Millisecond * 110)

	destRes = dest.Get()
	require.Equal(t, 2, len(destRes), "two comment notified")
	assert.Equal(t, "p2", destRes[1].Comment.ID)
	assert.Equal(t, "p1", destRes[1].parent.ID)
	assert.Equal(t, "u1", destRes[1].parent.User.ID)
	assert.Empty(t, destRes[1].Emails, "u1 is not notified they are the one who left the comment")

	// another reply to the first comment, another user
	s.Submit(Request{Comment: dataStore.data["p3"]})
	time.Sleep(time.Millisecond * 110)

	destRes = dest.Get()
	require.Equal(t, 3, len(destRes), "three comment notified")
	assert.Equal(t, "p3", destRes[2].Comment.ID)
	assert.Equal(t, "p1", destRes[2].parent.ID)
	assert.Equal(t, "u1", destRes[2].parent.User.ID)
	assert.ElementsMatch(t, []string{"u1@example.com"}, destRes[2].Emails)

	// reply to the last comment by another user, should trigger email retrieval error
	s.Submit(Request{Comment: dataStore.data["p4"]})
	time.Sleep(time.Millisecond * 110)

	destRes = dest.Get()
	require.Equal(t, 4, len(destRes), "four comment notified")
	assert.Equal(t, "p4", destRes[3].Comment.ID)
	assert.Equal(t, "p3", destRes[3].parent.ID)
	assert.Equal(t, "u2", destRes[3].parent.User.ID)
	assert.Empty(t, destRes[3].Emails, "no email can be retrieved for u2")

	s.Close()
}

func TestService_Recursive(t *testing.T) {
	dest := &MockDest{id: 1}
	dataStore := &mockStore{data: map[string]store.Comment{}, emailData: map[string]string{}}

	dataStore.data["p1"] = store.Comment{ID: "p1", User: store.User{ID: "u1"}}
	dataStore.data["p2"] = store.Comment{ID: "p2", ParentID: "p1", User: store.User{ID: "u2"}}
	dataStore.data["p3"] = store.Comment{ID: "p3", ParentID: "p2", User: store.User{ID: "u3"}}
	dataStore.data["p4"] = store.Comment{ID: "p4", ParentID: "p3", User: store.User{ID: "u1"}}
	dataStore.data["p5"] = store.Comment{ID: "p5", ParentID: "p4", User: store.User{ID: "u4"}}
	dataStore.emailData["u1"] = "u1@example.com"
	// second comment goes without email address for notification
	dataStore.emailData["u3"] = "u3@example.com"

	s := NewService(dataStore, 1, dest)
	assert.NotNil(t, s)

	// one comment from u1 with email set
	s.Submit(Request{Comment: dataStore.data["p1"]})
	time.Sleep(time.Millisecond * 110)

	destRes := dest.Get()
	require.Equal(t, 1, len(destRes), "one comment notified")
	assert.Equal(t, "p1", destRes[0].Comment.ID)
	assert.Empty(t, destRes[0].parent)
	assert.Empty(t, destRes[0].Emails)

	// reply to the first comment from u2 without email set
	s.Submit(Request{Comment: dataStore.data["p2"]})
	time.Sleep(time.Millisecond * 110)

	destRes = dest.Get()
	require.Equal(t, 2, len(destRes), "two comment notified")
	assert.Equal(t, "p2", destRes[1].Comment.ID)
	assert.Equal(t, "p1", destRes[1].parent.ID)
	assert.Equal(t, "u1", destRes[1].parent.User.ID)
	assert.ElementsMatch(t, []string{"u1@example.com"}, destRes[1].Emails)

	// reply to the second comment from u3 with email set
	s.Submit(Request{Comment: dataStore.data["p3"]})
	time.Sleep(time.Millisecond * 110)

	destRes = dest.Get()
	require.Equal(t, 3, len(destRes), "three comment notified")
	assert.Equal(t, "p3", destRes[2].Comment.ID)
	assert.Equal(t, "p2", destRes[2].parent.ID)
	assert.Equal(t, "u2", destRes[2].parent.User.ID)
	assert.ElementsMatch(t, []string{"u1@example.com"}, destRes[2].Emails)

	// reply to the third comment from u1 (author of the first comment), only u3 should be notified
	s.Submit(Request{Comment: dataStore.data["p4"]})
	time.Sleep(time.Millisecond * 110)

	destRes = dest.Get()
	require.Equal(t, 4, len(destRes), "four comment notified once each")
	assert.Equal(t, "p4", destRes[3].Comment.ID)
	assert.Equal(t, "p3", destRes[3].parent.ID)
	assert.Equal(t, "u3", destRes[3].parent.User.ID)
	assert.ElementsMatch(t, []string{"u3@example.com"}, destRes[3].Emails, "u1 is not notified they are the one who left the comment")

	// reply to the fourth comment from u4, u1 and u3 should be notified once as a result
	s.Submit(Request{Comment: dataStore.data["p5"]})
	time.Sleep(time.Millisecond * 110)

	destRes = dest.Get()
	require.Equal(t, 5, len(destRes), "four comment notified once each")
	assert.Equal(t, "p5", destRes[4].Comment.ID)
	assert.Equal(t, "p4", destRes[4].parent.ID)
	assert.Equal(t, "u1", destRes[4].parent.User.ID)
	assert.ElementsMatch(t, []string{"u1@example.com", "u3@example.com"}, destRes[4].Emails, "u3 and u1 notified once")

	s.Close()
}

func TestService_Nop(t *testing.T) {
	s := NopService
	s.Submit(Request{Comment: store.Comment{}})
	s.Close()
	assert.Equal(t, uint32(1), atomic.LoadUint32(&s.closed))
}

type mockStore struct {
	data      map[string]store.Comment
	emailData map[string]string
}

func (m mockStore) Get(_ store.Locator, id string, _ store.User) (store.Comment, error) {
	res, ok := m.data[id]
	if !ok {
		return store.Comment{}, errors.New("no such id")
	}
	return res, nil
}

func (m mockStore) GetUserEmail(_, userID string) (string, error) {
	email, ok := m.emailData[userID]
	if !ok {
		return "", errors.New("no such user")
	}
	return email, nil
}
