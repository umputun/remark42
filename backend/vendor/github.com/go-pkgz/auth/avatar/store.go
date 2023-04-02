package avatar

import (
	"context"
	"crypto/sha1" //nolint gosec
	"encoding/hex"
	"fmt"
	_ "image/gif"  // initializing packages for supporting GIF
	_ "image/jpeg" // initializing packages for supporting JPEG.
	_ "image/png"  // initializing packages for supporting PNG.
	"io"
	"log"
	"net/url"
	"regexp"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/go-pkgz/auth/token"
)

// imgSfx for avatars
const imgSfx = ".image"

var reValidAvatarID = regexp.MustCompile(`^[a-fA-F0-9]{40}\.image$`)

// Store defines interface to store and load avatars
type Store interface {
	fmt.Stringer
	Put(userID string, reader io.Reader) (avatarID string, err error) // save avatar data from the reader and return base name
	Get(avatarID string) (reader io.ReadCloser, size int, err error)  // load avatar via reader
	ID(avatarID string) (id string)                                   // unique id of stored avatar's data
	Remove(avatarID string) error                                     // remove avatar data
	List() (ids []string, err error)                                  // list all avatar ids
	Close() error                                                     // close store
}

// NewStore provides factory for all supported stores making the one
// based on uri protocol. Default (no protocol) is file-system
func NewStore(uri string) (Store, error) {
	switch {
	case strings.HasPrefix(uri, "file://"):
		return NewLocalFS(strings.TrimPrefix(uri, "file://")), nil
	case !strings.Contains(uri, "://"):
		return NewLocalFS(uri), nil
	case strings.HasPrefix(uri, "mongodb://"), strings.HasPrefix(uri, "mongodb+srv://"):
		db, bucketName, u, err := parseExtMongoURI(uri)
		if err != nil {
			return nil, fmt.Errorf("can't parse mongo store uri %s: %w", uri, err)
		}

		const timeout = time.Second * 30
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		client, err := mongo.Connect(ctx, options.Client().ApplyURI(u).SetConnectTimeout(timeout))
		if err != nil {
			return nil, fmt.Errorf("failed to connect to mongo server: %w", err)
		}
		if err = client.Ping(ctx, nil); err != nil {
			return nil, fmt.Errorf("failed to connect to mongo server: %w", err)
		}
		return NewGridFS(client, db, bucketName, time.Second*5), nil
	case strings.HasPrefix(uri, "bolt://"):
		return NewBoltDB(strings.TrimPrefix(uri, "bolt://"), bolt.Options{})
	}
	return nil, fmt.Errorf("can't parse store url %s", uri)
}

// Migrate avatars between stores
func Migrate(dst, src Store) (int, error) {
	ids, err := src.List()
	if err != nil {
		return 0, err
	}
	for _, id := range ids {
		srcReader, _, err := src.Get(id)
		if err != nil {
			log.Printf("[WARN] can't get reader for avatar %s", id)
			continue
		}
		if _, err = dst.Put(id, srcReader); err != nil {
			log.Printf("[WARN] can't put avatar %s", id)
		}
		if err = srcReader.Close(); err != nil {
			log.Printf("[WARN] failed to close avatar %s", id)
		}
	}
	return len(ids), nil
}

// encodeID hashes id to sha1. Skip encoding for already processed
func encodeID(id string) string {
	if reValidAvatarID.MatchString(id) {
		return strings.TrimSuffix(id, imgSfx) // already encoded, strip .image
	}
	return token.HashID(sha1.New(), id)
}

// parseExtMongoURI extracts extra params ava_db and ava_coll and remove
// from the url. Input example: mongodb://user:password@127.0.0.1:27017/test?ssl=true&ava_db=db1&ava_coll=coll1
func parseExtMongoURI(uri string) (db, collection, cleanURI string, err error) {

	db, collection = "test", "avatars_fs"
	u, err := url.Parse(uri)
	if err != nil {
		return "", "", "", err
	}
	if val := u.Query().Get("ava_db"); val != "" {
		db = val
	}
	if val := u.Query().Get("ava_coll"); val != "" {
		collection = val
	}

	q := u.Query()
	q.Del("ava_db")
	q.Del("ava_coll")
	u.RawQuery = q.Encode()
	return db, collection, u.String(), nil
}

func hash(data []byte, avatarID string) (id string) {
	h := sha1.New()
	if _, err := h.Write(data); err != nil {
		log.Printf("[DEBUG] can't apply sha1 for content of '%s', %s", avatarID, err)
		return encodeID(avatarID)
	}
	return hex.EncodeToString(h.Sum(nil))
}
