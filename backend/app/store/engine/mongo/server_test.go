package mongo

import (
	"testing"
	"time"

	"github.com/globalsign/mgo"
	"github.com/stretchr/testify/assert"
)

func TestNewServerGood(t *testing.T) {
	m, err := NewServer(mgo.DialInfo{Addrs: []string{"mongo"}}, ServerParams{})
	assert.Nil(t, err)
	assert.NotNil(t, m)

	m, err = NewServer(mgo.DialInfo{Addrs: []string{"mongo"}}, ServerParams{Debug: true})
	assert.Nil(t, err)
	assert.NotNil(t, m)

	st := time.Now()
	m, err = NewServer(mgo.DialInfo{Addrs: []string{"mongo"}}, ServerParams{Debug: true, Delay: 1})
	assert.Nil(t, err)
	assert.NotNil(t, m)
	assert.True(t, time.Now().After(st.Add(time.Millisecond*999)), "should take a second")

}

func TestNewServerBad(t *testing.T) {
	_, err := NewServer(mgo.DialInfo{Addrs: []string{"127.0.0.2"}, Timeout: time.Second}, ServerParams{})
	assert.NotNil(t, err)

	_, err = NewServer(mgo.DialInfo{}, ServerParams{})
	assert.NotNil(t, err)
}
