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

// Put avatar to gridfs object, try to resize
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
	_, err = bucket.UploadFromStream(id+imgSfx, buf, &options.UploadOptions{Metadata: bson.M{"hash": avaHash}})
	return id + imgSfx, err
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

// ID returns a fingerprint of the avatar content. Uses MD5 because gridfs provides it directly
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
	cursor, err := bucket.Find(bson.M{"filename": avatar})
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
	cursor, err := bucket.Find(bson.M{"filename": avatar})
	if err != nil {
		return err
	}

	r := struct {
		ID primitive.ObjectID `bson:"_id"`
	}{}
	ctx, cancel := context.WithTimeout(context.Background(), gf.timeout)
	defer cancel()
	if found := cursor.Next(ctx); found {
		if err := cursor.Decode(&r); err != nil {
			return err
		}
		return bucket.Delete(r.ID)
	}
	return fmt.Errorf("avatar %s not found", avatar)
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
