package avatar

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"

	"github.com/globalsign/mgo"
	"github.com/pkg/errors"

	"github.com/umputun/remark/backend/app/store"
	"github.com/umputun/remark/backend/app/store/engine/mongo"
)

// NewGridFS makes gridfs (mongo) avatar store
func NewGridFS(conn *mongo.Connection, resizeLimit int) *GridFS {
	return &GridFS{Connection: conn, resizeLimit: resizeLimit}
}

// GridFS implements Store for GridFS
type GridFS struct {
	Connection  *mongo.Connection
	resizeLimit int
}

// Put avatear to gridfs object, try to resize
func (gf *GridFS) Put(userID string, reader io.Reader) (avatar string, err error) {
	id := store.EncodeID(userID)
	err = gf.Connection.WithDB(func(dbase *mgo.Database) error {
		fh, e := dbase.GridFS("fs").Create(id + imgSfx)
		if e != nil {
			return e
		}
		defer func() {
			if err = fh.Close(); err != nil {
				log.Printf("[WARN] can't close avatar file %v, %s", fh, err)
			}
		}()

		// Trying to resize avatar.
		if reader = resize(reader, gf.resizeLimit); reader == nil {
			return errors.New("avatar reader is nil")
		}
		_, e = io.Copy(fh, reader)
		return e
	})
	return id + imgSfx, err
}

// Get avatar reader for avatar id.image
func (gf *GridFS) Get(avatar string) (reader io.ReadCloser, size int, err error) {
	buf := &bytes.Buffer{}
	err = gf.Connection.WithDB(func(dbase *mgo.Database) error {
		fh, e := dbase.GridFS("fs").Open(avatar)
		if e != nil {
			return errors.Wrapf(e, "can't load avatar %s", avatar)
		}
		if _, e = io.Copy(buf, fh); e != nil {
			return errors.Wrapf(e, "can't copy avatar %s", avatar)
		}
		size = int(fh.Size())
		return fh.Close()
	})
	return ioutil.NopCloser(buf), size, err
}

// ID returns a fingerprint of the avatar content. Uses MD5 because gridfs provides it directly
func (gf *GridFS) ID(avatar string) (id string) {
	err := gf.Connection.WithDB(func(dbase *mgo.Database) error {
		fh, e := dbase.GridFS("fs").Open(avatar)
		if e != nil {
			return errors.Wrapf(e, "can't open avatar %s", avatar)
		}
		id = fh.MD5()
		return nil
	})
	if err != nil {
		log.Printf("[DEBUG] can't get file info '%s', %s", avatar, err)
		return store.EncodeID(avatar)
	}
	return id
}
