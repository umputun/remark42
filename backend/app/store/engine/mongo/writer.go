package mongo

import (
	"sync"
	"time"

	"github.com/globalsign/mgo"
	"github.com/pkg/errors"
)

// BufferedWriter defines interface for writes and flush
type BufferedWriter interface {
	Write(rec interface{}) error
	Flush() error
}

// BufferedWriterMgo collects records in local buffer and flushes them as filled. Thread safe
// by default using both DB and collection from provided connection.
// Collection can be customized by WithCollection method. Optional flush duration to save on interval
type BufferedWriterMgo struct {
	connection    *Connection
	bufferSize    int
	collection    string
	flushDuration time.Duration

	buffer        []interface{}
	lock          sync.Mutex
	lastWriteTime time.Time
	once          sync.Once
}

// NewBufferedWriter makes batch writer for given size and connection
func NewBufferedWriter(size int, connection *Connection) *BufferedWriterMgo {
	if size == 0 {
		size = 1
	}
	return &BufferedWriterMgo{
		bufferSize: size,
		buffer:     make([]interface{}, 0, size+1),
		connection: connection,
	}
}

// WithCollection sets custom collection to use with writer
func (bw *BufferedWriterMgo) WithCollection(collection string) *BufferedWriterMgo {
	bw.collection = collection
	return bw
}

// WithAutoFlush sets auto flush duration
func (bw *BufferedWriterMgo) WithAutoFlush(duration time.Duration) *BufferedWriterMgo {
	bw.flushDuration = duration
	if duration > 0 { // activate background auto-flush
		bw.once.Do(func() {
			ticker := time.NewTicker(duration)
			go func() {
				for range ticker.C {
					shouldFlush := false
					bw.synced(func() {
						shouldFlush = bw.lastWriteTime.Before(time.Now().Add(-1*bw.flushDuration)) && len(bw.buffer) > 0
					})
					if shouldFlush {
						bw.Flush()
					}
				}
			}()
		})
	}
	return bw
}

// Write to buffer and, as filled, to mongo. If flushDuration defined check for automatic flush
func (bw *BufferedWriterMgo) Write(rec interface{}) error {
	bw.lock.Lock()
	defer bw.lock.Unlock()

	bw.lastWriteTime = time.Now()
	bw.buffer = append(bw.buffer, rec)
	if len(bw.buffer) >= bw.bufferSize {
		err := bw.writeBuffer()
		bw.buffer = bw.buffer[0:0]
		return errors.Wrapf(err, "failed to write to %s", bw.connection)
	}
	return nil
}

// Flush writes everything left in buffer to mongo
func (bw *BufferedWriterMgo) Flush() error {
	bw.lock.Lock()
	defer bw.lock.Unlock()

	if len(bw.buffer) > 0 {
		err := bw.writeBuffer()
		bw.buffer = bw.buffer[0:0]
		return errors.Wrapf(err, "failed to flush to %s", bw.connection)
	}
	return nil
}

// writeBuffer sends all collected records to mongo
func (bw *BufferedWriterMgo) writeBuffer() (err error) {

	if bw.collection == "" { // no custom collection
		err = bw.connection.WithCollection(func(coll *mgo.Collection) error {
			return coll.Insert(bw.buffer...)
		})
	}

	if bw.collection != "" { // with custom collection
		err = bw.connection.WithCustomCollection(bw.collection, func(coll *mgo.Collection) error {
			return coll.Insert(bw.buffer...)
		})
	}

	return err
}

func (bw *BufferedWriterMgo) synced(fn func()) {
	bw.lock.Lock()
	fn()
	bw.lock.Unlock()
}
