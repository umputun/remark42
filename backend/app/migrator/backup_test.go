package migrator

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackup_RemoveOldBackupFiles(t *testing.T) {
	loc := "/tmp/remark-backups.test"
	defer os.RemoveAll(loc)

	assert.NoError(t, os.MkdirAll(loc, 0o700))

	for i := 1; i <= 10; i++ {
		fname := fmt.Sprintf("%s/backup-site1-201712%02d.gz", loc, i)
		err := os.WriteFile(fname, []byte("blah"), 0o600)
		assert.NoError(t, err)
	}
	fname := fmt.Sprintf("%s/backup-site2-20171210.gz", loc)
	err := os.WriteFile(fname, []byte("blah"), 0o600)
	assert.NoError(t, err)

	bk := AutoBackup{BackupLocation: loc, SiteID: "site1", KeepMax: 3}
	bk.removeOldBackupFiles()
	ff, err := os.ReadDir(loc)
	assert.NoError(t, err)
	require.Equal(t, 4, len(ff), "should keep 4 files - 3 kept for sit1, and one for site2")
	assert.Equal(t, "backup-site1-20171208.gz", ff[0].Name())
	assert.Equal(t, "backup-site1-20171209.gz", ff[1].Name())
	assert.Equal(t, "backup-site1-20171210.gz", ff[2].Name())
	assert.Equal(t, "backup-site2-20171210.gz", ff[3].Name())
}

func TestBackup_MakeBackup(t *testing.T) {
	loc := "/tmp/remark-backups.test"
	defer os.RemoveAll(loc)
	assert.NoError(t, os.MkdirAll(loc, 0o700))

	bk := AutoBackup{BackupLocation: loc, SiteID: "site1", KeepMax: 3, Exporter: &mockExporter{}}
	fname, err := bk.makeBackup()
	assert.NoError(t, err)
	expFile := fmt.Sprintf("/tmp/remark-backups.test/backup-site1-%s.gz", time.Now().Format("20060102"))
	assert.Equal(t, expFile, fname)

	assert.Equal(t, exportedPayload, gzContent(t, expFile))
}

func TestBackup_Do(t *testing.T) {
	loc := "/tmp/remark-backups.test"
	defer os.RemoveAll(loc)
	assert.NoError(t, os.MkdirAll(loc, 0o700))

	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(time.Second)
			cancel()
		}()

		bk := AutoBackup{BackupLocation: loc, SiteID: "site1", KeepMax: 3, Exporter: &mockExporter{}, Duration: 600 * time.Millisecond}
		bk.Do(ctx)

		expFile := fmt.Sprintf("/tmp/remark-backups.test/backup-site1-%s.gz", time.Now().Format("20060102"))
		assert.Equal(t, exportedPayload, gzContent(t, expFile))
	})
}

const exportedPayload = "some export blah blah 1234567890"

// the compressed size is not assertable: it moves with the compress/flate version
func gzContent(t *testing.T, name string) string {
	t.Helper()
	fh, err := os.Open(name) //nolint:gosec // path is built by the test
	require.NoError(t, err)
	defer func() { assert.NoError(t, fh.Close()) }()

	gz, err := gzip.NewReader(fh)
	require.NoError(t, err)
	defer func() { assert.NoError(t, gz.Close()) }()

	b, err := io.ReadAll(gz)
	require.NoError(t, err)
	return string(b)
}

type mockExporter struct{}

func (mock *mockExporter) Export(w io.Writer, _ string) (int, error) {
	_, err := w.Write([]byte(exportedPayload))
	return 1000, err
}
