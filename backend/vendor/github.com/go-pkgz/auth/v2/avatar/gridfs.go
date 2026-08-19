package avatar

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// NewGridFS makes gridfs (mongo) avatar store
func NewGridFS(client *mongo.Client, dbName, bucketName string, timeout time.Duration) *GridFS {
	return &GridFS{client: client, db: client.Database(dbName), bucketName: bucketName, timeout: timeout}
}

// GridFS implements Store for GridFS
type GridFS struct {
	client     *mongo.Client
	db         *mongo.Database
	bucketName string
	timeout    time.Duration
}

// Put stores avatar bytes read from reader in GridFS, keyed by the encoded userID
// with .image suffix, and records the sha1 of those bytes in the file's metadata.
// Resizing happens upstream in Proxy.resize; this layer only writes what it is given.
func (gf *GridFS) Put(userID string, reader io.Reader) (avatar string, err error) {
	id := encodeID(userID)
	bucket, err := gridfs.NewBucket(gf.db, &options.BucketOptions{Name: &gf.bucketName})
	if err != nil {
		return "", err
	}

	buf := &bytes.Buffer{}
	if _, err = io.Copy(buf, reader); err != nil {
		return "", fmt.Errorf("can't read avatar for %s: %w", userID, err)
	}

	avaHash := hash(buf.Bytes(), id)
	fileID, err := bucket.UploadFromStream(id+imgSfx, buf, &options.UploadOptions{Metadata: bson.M{"hash": avaHash}})
	if err != nil {
		return "", err
	}

	// gridfs turns every upload with the same name into another revision, so drop what
	// earlier Put calls left behind. Cleanup runs after the upload, keeping the previous
	// avatar in place if the write failed, and is best-effort as the new avatar is stored
	// either way.
	_ = gf.removeOlderRevisions(bucket, id+imgSfx, fileID)

	return id + imgSfx, nil
}

// removeOlderRevisions deletes the revisions of fileName stored before keepID. Only older
// revisions go, so two concurrent Put calls cannot delete each other's upload and leave the
// avatar missing; the newer of the two survives.
func (gf *GridFS) removeOlderRevisions(bucket *gridfs.Bucket, fileName string, keepID primitive.ObjectID) error {
	ids, err := gf.revisionIDs(bucket, fileName)
	if err != nil {
		return err
	}

	older := false
	for _, id := range ids { // newest first, everything past keepID was uploaded earlier
		if id == keepID {
			older = true
			continue
		}
		if !older {
			continue
		}
		if e := bucket.Delete(id); e != nil {
			err = e
		}
	}
	return err
}

// revisionIDs returns ids of all gridfs files stored under the given name, newest first
func (gf *GridFS) revisionIDs(bucket *gridfs.Bucket, fileName string) ([]primitive.ObjectID, error) {
	sortNewestFirst := options.GridFSFind().SetSort(bson.D{{Key: "uploadDate", Value: -1}, {Key: "_id", Value: -1}})
	cursor, err := bucket.Find(bson.M{"filename": fileName}, sortNewestFirst)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), gf.timeout)
	defer cancel()

	var ids []primitive.ObjectID
	for cursor.Next(ctx) {
		r := struct {
			ID primitive.ObjectID `bson:"_id"`
		}{}
		if err = cursor.Decode(&r); err != nil {
			return nil, err
		}
		ids = append(ids, r.ID)
	}
	return ids, cursor.Err()
}

// Get avatar reader for avatar id.image
func (gf *GridFS) Get(avatar string) (reader io.ReadCloser, size int, err error) {
	bucket, err := gridfs.NewBucket(gf.db, &options.BucketOptions{Name: &gf.bucketName})
	if err != nil {
		return nil, 0, err
	}
	buf := &bytes.Buffer{}
	sz, e := bucket.DownloadToStreamByName(avatar, buf)
	if e != nil {
		return nil, 0, fmt.Errorf("can't read avatar %s: %w", avatar, e)
	}
	return io.NopCloser(buf), int(sz), nil
}

// ID returns the sha1 fingerprint of the avatar content stored in the GridFS file's
// metadata at Put time, or an encoded fallback id when the file lookup/decode fails.
func (gf *GridFS) ID(avatar string) (id string) {

	finfo := struct {
		ID       primitive.ObjectID `bson:"_id"`
		Len      int                `bson:"length"`
		FileName string             `bson:"filename"`
		MetaData struct {
			Hash string `bson:"hash"`
		} `bson:"metadata"`
	}{}

	bucket, err := gridfs.NewBucket(gf.db, &options.BucketOptions{Name: &gf.bucketName})
	if err != nil {
		return encodeID(avatar)
	}
	sortNewestFirst := options.GridFSFind().SetSort(bson.D{{Key: "uploadDate", Value: -1}, {Key: "_id", Value: -1}})
	cursor, err := bucket.Find(bson.M{"filename": avatar}, sortNewestFirst)
	if err != nil {
		return encodeID(avatar)
	}

	ctx, cancel := context.WithTimeout(context.Background(), gf.timeout)
	defer cancel()
	if found := cursor.Next(ctx); found {
		if err = cursor.Decode(&finfo); err != nil {
			return encodeID(avatar)
		}
		return finfo.MetaData.Hash
	}
	return encodeID(avatar)
}

// Remove avatar from gridfs
func (gf *GridFS) Remove(avatar string) error {
	bucket, err := gridfs.NewBucket(gf.db, &options.BucketOptions{Name: &gf.bucketName})
	if err != nil {
		return err
	}
	ids, err := gf.revisionIDs(bucket, avatar)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return fmt.Errorf("avatar %s not found: %w", avatar, ErrNotFound)
	}

	// every revision has to go, deleting the newest one alone leaves the avatar readable
	for _, id := range ids {
		if e := bucket.Delete(id); e != nil {
			err = e
		}
	}
	return err
}

// List all avatars (ids) on gfs
// note: id includes .image suffix
func (gf *GridFS) List() (ids []string, err error) {
	bucket, err := gridfs.NewBucket(gf.db, &options.BucketOptions{Name: &gf.bucketName})
	if err != nil {
		return nil, err
	}

	gfsFile := struct {
		Filename string `bson:"filename,omitempty"`
	}{}
	cursor, err := bucket.Find(bson.M{})
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), gf.timeout)
	defer cancel()
	for cursor.Next(ctx) {
		if err := cursor.Decode(&gfsFile); err != nil {
			return nil, err
		}
		ids = append(ids, gfsFile.Filename)
	}
	return ids, nil
}

// Close gridfs store
func (gf *GridFS) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), gf.timeout)
	defer cancel()
	if err := gf.client.Disconnect(ctx); err != nil && err != mongo.ErrClientDisconnected {
		return err
	}
	return nil
}

func (gf *GridFS) String() string {
	return fmt.Sprintf("mongo (grid fs), db=%s, bucket=%s", gf.db.Name(), gf.bucketName)
}
