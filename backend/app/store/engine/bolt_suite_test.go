package engine_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"

	"github.com/umputun/remark42/backend/app/store/engine"
	"github.com/umputun/remark42/backend/app/store/engine/enginetest"
)

// TestBoltDB_Engine runs the shared engine test set. It sits in package engine_test, apart from bolt_test.go,
// because enginetest imports engine and a test of package engine itself can't import it back.
func TestBoltDB_Engine(t *testing.T) {
	enginetest.Run(t, func(t *testing.T) engine.Interface {
		dir := t.TempDir()
		b, err := engine.NewBoltDB(bolt.Options{},
			engine.BoltSite{FileName: filepath.Join(dir, "site.db"), SiteID: enginetest.SiteID},
			engine.BoltSite{FileName: filepath.Join(dir, "other-site.db"), SiteID: enginetest.OtherSiteID})
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, b.Close()) })
		return b
	})
}
