package mongo

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/globalsign/mgo"
)

// BufferedWriter defines interface for writes and flush
type BufferedWriter interface {
	Write(rec interface{}) error
	Flush() error
	Close() error
}

// BufferedWriterMgo collects records in local buffer and flushes them as filled. Thread safe
// by default using both DB and collection from provided connection.
// Collection can be customized by WithCollection method. Optional flush duration to save on interval
type BufferedWriterMgo struct {
	connection    *Connection
	bufferSize    int
	collection    string
	flushDuration time.Duration

	ctx    context.Context
	cancel context.CancelFunc

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
			bw.ctx, bw.cancel = context.WithCancel(context.Background())
			ticker := time.NewTicker(duration)
			go func() {
				defer bw.cancel()
				for {
					select {
					case <-ticker.C:
						var shouldFlush bool
						_ = bw.synced(func() error {
							shouldFlush = time.Now().After(bw.lastWriteTime.Add(bw.flushDuration)) && len(bw.buffer) > 0
							return nil
						})
						if shouldFlush {
							if err := bw.Flush(); err != nil {
								log.Printf("[WARN] flush failed, %s", err)
							}
						}
					case <-bw.ctx.Done():
						return
					}
				}
			}()
		})
	}
	return bw
}

// Write to buffer and, as filled, to mongo. If flushDuration defined check for automatic flush
func (bw *BufferedWriterMgo) Write(rec interface{}) error {
	return bw.synced(func() error {
		bw.lastWriteTime = time.Now()
		bw.buffer = append(bw.buffer, rec)
		if len(bw.buffer) >= bw.bufferSize {
			if err := bw.writeBuffer(); err != nil {
				return fmt.Errorf("failed to write to %s, %s", bw.connection, err)
			}
			bw.buffer = bw.buffer[0:0]
		}
		return nil
	})
}

// Flush writes everything left in buffer to mongo
func (bw *BufferedWriterMgo) Flush() error {
	err := bw.synced(func() error {
		err := bw.writeBuffer()
		bw.buffer = bw.buffer[0:0]
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to flush to %s, %s", bw.connection, err)
	}
	return nil
}

// Close flushes all in-fly records and terminates background auto-flusher
func (bw *BufferedWriterMgo) Close() (err error) {
	return bw.synced(func() error {
		err = bw.writeBuffer()
		if bw.flushDuration > 0 {
			bw.cancel()
			<-bw.ctx.Done()
		}
		return err
	})
}

// writeBuffer sends all collected records to mongo
func (bw *BufferedWriterMgo) writeBuffer() (err error) {

	if len(bw.buffer) == 0 {
		return nil
	}

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

func (bw *BufferedWriterMgo) synced(fn func() error) error {
	bw.lock.Lock()
	defer bw.lock.Unlock()
	return fn()
}
