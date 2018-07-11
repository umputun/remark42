package mongo

import (
	"sync"
	"testing"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/stretchr/testify/assert"
)

func TestWriter(t *testing.T) {

	count := func(conn *Connection) (res int) {
		_ = conn.WithCollection(func(coll *mgo.Collection) error {
			var err error
			res, err = coll.Find(nil).Count()
			assert.Nil(t, err)
			return nil
		})
		return res
	}

	conn := MakeTestConnection(t)
	defer RemoveTestCollection(t, conn)

	var wr BufferedWriter = NewBufferedWriter(3, conn)
	assert.Nil(t, wr.Write(bson.M{"key1": "val1"}), "write rec #1")
	assert.Nil(t, wr.Write(bson.M{"key2": "val2"}), "write rec #2")

	assert.Equal(t, 0, count(conn), "nothing yet")

	assert.Nil(t, wr.Write(bson.M{"key3": "val3"}), "write rec #3")
	assert.Equal(t, 3, count(conn), "all 3 records in")

	assert.Nil(t, wr.Write(bson.M{"key4": "val4"}), "write rec #4")
	assert.Equal(t, 3, count(conn), "still 3 records")

	assert.Nil(t, wr.Flush())
	assert.Equal(t, 4, count(conn), "all 4 records")

	assert.Nil(t, wr.Flush())
	assert.Equal(t, 4, count(conn), "still 4 records, nothing left to flush")

	assert.Nil(t, wr.Close())

}

func TestWriterParallel(t *testing.T) {
	conn := MakeTestConnection(t)
	defer RemoveTestCollection(t, conn)

	var wg sync.WaitGroup
	wr := NewBufferedWriter(75, conn)

	writeMany := func() {
		for i := 0; i < 100; i++ {
			wr.Write(bson.M{"key1": 1, "key2": 2})
		}
		wr.Flush()
		wg.Done()
	}

	for i := 0; i < 16; i++ {
		wg.Add(1)
		go writeMany()
	}

	wg.Wait()

	_ = conn.WithCollection(func(coll *mgo.Collection) error {
		res, err := coll.Find(nil).Count()
		assert.Nil(t, err)
		assert.Equal(t, 100*16, res)
		return nil
	})
	assert.Nil(t, wr.Close())
}

func TestWriterWithAuthFlush(t *testing.T) {

	conn := MakeTestConnection(t)
	defer RemoveTestCollection(t, conn)

	var wr BufferedWriter = NewBufferedWriter(3, conn).WithAutoFlush(500 * time.Millisecond)
	count := func() (res int) {
		_ = conn.WithCollection(func(coll *mgo.Collection) error {
			var err error
			res, err = coll.Find(nil).Count()
			assert.Nil(t, err)
			return nil
		})
		return res
	}

	assert.Nil(t, wr.Write(bson.M{"key1": "val1"}), "write rec #1")
	assert.Nil(t, wr.Write(bson.M{"key2": "val2"}), "write rec #2")
	assert.Equal(t, 0, count(), "nothing yet")
	time.Sleep(600 * time.Millisecond)
	assert.Equal(t, 2, count(), "2 records flushed")

	assert.Nil(t, wr.Write(bson.M{"key3": "val3"}), "write rec #3")
	assert.Nil(t, wr.Write(bson.M{"key4": "val4"}), "write rec #4")
	assert.Nil(t, wr.Write(bson.M{"key5": "val5"}), "write rec #5")
	assert.Equal(t, 5, count(), "5 records, flushed by size, not duration")

	assert.Nil(t, wr.Write(bson.M{"key6": "val6"}), "write rec #6")
	assert.Nil(t, wr.Write(bson.M{"key7": "val7"}), "write rec #7")
	assert.Equal(t, 5, count(), "still 5 records")

	assert.Nil(t, wr.Flush())
	assert.Equal(t, 7, count(), "all 7 records")

	assert.Nil(t, wr.Flush())
	assert.Equal(t, 7, count(), "still 7 records, nothing left to flush")
	assert.Nil(t, wr.Close())
}

func TestWriterParallelWithAutoFlush(t *testing.T) {
	conn := MakeTestConnection(t)
	defer RemoveTestCollection(t, conn)

	var wg sync.WaitGroup
	wr := NewBufferedWriter(75, conn).WithAutoFlush(time.Millisecond)

	writeMany := func() {
		for i := 0; i < 100; i++ {
			wr.Write(bson.M{"key1": 1, "key2": 2})
			time.Sleep(time.Millisecond * 3)
		}
		wr.Flush()
		wg.Done()
	}

	for i := 0; i < 16; i++ {
		wg.Add(1)
		go writeMany()
	}

	wg.Wait()

	_ = conn.WithCollection(func(coll *mgo.Collection) error {
		res, err := coll.Find(nil).Count()
		assert.Nil(t, err)
		assert.Equal(t, 100*16, res)
		return nil
	})
	assert.Nil(t, wr.Close())
}
