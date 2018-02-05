package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStore_sortComments(t *testing.T) {
	cc := []Comment{
		{ID: "1", Score: 5, Timestamp: time.Date(2018, 2, 5, 10, 1, 0, 0, time.Local)},
		{ID: "2", Score: 4, Timestamp: time.Date(2018, 2, 5, 10, 2, 0, 0, time.Local)},
		{ID: "3", Score: 6, Timestamp: time.Date(2018, 2, 5, 10, 3, 0, 0, time.Local)},
	}

	sortComments(cc, "+time")
	assert.Equal(t, "1", cc[0].ID)
	assert.Equal(t, "2", cc[1].ID)
	assert.Equal(t, "3", cc[2].ID)

	sortComments(cc, "-time")
	assert.Equal(t, "3", cc[0].ID)
	assert.Equal(t, "2", cc[1].ID)
	assert.Equal(t, "1", cc[2].ID)

	sortComments(cc, "score")
	assert.Equal(t, "2", cc[0].ID)
	assert.Equal(t, "1", cc[1].ID)
	assert.Equal(t, "3", cc[2].ID)

	sortComments(cc, "-score")
	assert.Equal(t, "3", cc[0].ID)
	assert.Equal(t, "1", cc[1].ID)
	assert.Equal(t, "2", cc[2].ID)
}
