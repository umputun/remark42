package engine

import (
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/umputun/remark42/backend/app/store"
)

func TestEngine_sortComments(t *testing.T) {
	cc := []store.Comment{
		{ID: "1", Score: 5, Controversy: 1, Timestamp: time.Date(2018, 2, 5, 10, 1, 0, 0, time.UTC)},
		{ID: "2", Score: 4, Controversy: 2, Timestamp: time.Date(2018, 2, 5, 10, 2, 0, 0, time.UTC)},
		{ID: "3", Score: 6, Controversy: 3, Timestamp: time.Date(2018, 2, 5, 10, 3, 0, 0, time.UTC)},
		{ID: "4", Score: 6, Controversy: 1, Timestamp: time.Date(2018, 2, 5, 10, 4, 0, 0, time.UTC)},
	}

	SortComments(cc, "+time")
	assert.Equal(t, "1", cc[0].ID)
	assert.Equal(t, "2", cc[1].ID)
	assert.Equal(t, "3", cc[2].ID)
	assert.Equal(t, "4", cc[3].ID)

	SortComments(cc, "-time")
	assert.Equal(t, "4", cc[0].ID)
	assert.Equal(t, "3", cc[1].ID)
	assert.Equal(t, "2", cc[2].ID)
	assert.Equal(t, "1", cc[3].ID)

	SortComments(cc, "score")
	assert.Equal(t, "2", cc[0].ID)
	assert.Equal(t, "1", cc[1].ID)
	assert.Equal(t, "3", cc[2].ID)
	assert.Equal(t, "4", cc[3].ID)

	SortComments(cc, "-score")
	assert.Equal(t, "3", cc[0].ID)
	assert.Equal(t, "4", cc[1].ID)
	assert.Equal(t, "1", cc[2].ID)
	assert.Equal(t, "2", cc[3].ID)

	SortComments(cc, "controversy")
	assert.Equal(t, "1", cc[0].ID)
	assert.Equal(t, "4", cc[1].ID)
	assert.Equal(t, "2", cc[2].ID)
	assert.Equal(t, "3", cc[3].ID)

	SortComments(cc, "-controversy")
	assert.Equal(t, "3", cc[0].ID)
	assert.Equal(t, "2", cc[1].ID)
	assert.Equal(t, "1", cc[2].ID)
	assert.Equal(t, "4", cc[3].ID)
}

func TestEngine_sortCommentsUnknownField(t *testing.T) {
	// the time order, 2 3 1, differs from the input order and from both orders by score and by controversy
	unsorted := []store.Comment{
		{ID: "1", Score: 5, Controversy: 2, Timestamp: time.Date(2018, 2, 5, 10, 3, 0, 0, time.UTC)},
		{ID: "2", Score: 6, Controversy: 3, Timestamp: time.Date(2018, 2, 5, 10, 1, 0, 0, time.UTC)},
		{ID: "3", Score: 4, Controversy: 1, Timestamp: time.Date(2018, 2, 5, 10, 2, 0, 0, time.UTC)},
	}
	// anything but the listed values with at most one sign sorts by time, oldest first
	for _, sortFld := range []string{"", "bogus", "-bogus", "--score", "-+time", "+-active", "Score", " -score",
		"Controversy", "controversy "} {
		cc := SortComments(slices.Clone(unsorted), sortFld)
		assert.Equal(t, []string{"2", "3", "1"}, []string{cc[0].ID, cc[1].ID, cc[2].ID}, "sort %q", sortFld)
	}
}
