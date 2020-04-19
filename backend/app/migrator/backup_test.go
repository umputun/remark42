package migrator

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackup_RemoveOldBackupFiles(t *testing.T) {
	loc := "/tmp/remark-backups.test"
	defer os.RemoveAll(loc)

	assert.NoError(t, os.MkdirAll(loc, 0700))

	for i := 1; i <= 10; i++ {
		fname := fmt.Sprintf("%s/backup-site1-201712%02d.gz", loc, i)
		err := ioutil.WriteFile(fname, []byte("blah"), 0600)
		assert.NoError(t, err)
	}
	fname := fmt.Sprintf("%s/backup-site2-20171210.gz", loc)
	err := ioutil.WriteFile(fname, []byte("blah"), 0600)
	assert.NoError(t, err)

	bk := AutoBackup{BackupLocation: loc, SiteID: "site1", KeepMax: 3}
	bk.removeOldBackupFiles()
	ff, err := ioutil.ReadDir(loc)
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
	assert.NoError(t, os.MkdirAll(loc, 0700))

	bk := AutoBackup{BackupLocation: loc, SiteID: "site1", KeepMax: 3, Exporter: &mockExporter{}}
	fname, err := bk.makeBackup()
	assert.NoError(t, err)
	expFile := fmt.Sprintf("/tmp/remark-backups.test/backup-site1-%s.gz", time.Now().Format("20060102"))
	assert.Equal(t, expFile, fname)

	fi, err := os.Lstat(expFile)
	assert.NoError(t, err)
	assert.Equal(t, int64(52), fi.Size())
}

func TestBackup_Do(t *testing.T) {
	loc := "/tmp/remark-backups.test"
	defer os.RemoveAll(loc)
	assert.NoError(t, os.MkdirAll(loc, 0700))

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(time.Second)
		cancel()
	}()

	bk := AutoBackup{BackupLocation: loc, SiteID: "site1", KeepMax: 3, Exporter: &mockExporter{}, Duration: 600 * time.Millisecond}
	bk.Do(ctx)

	expFile := fmt.Sprintf("/tmp/remark-backups.test/backup-site1-%s.gz", time.Now().Format("20060102"))
	fi, err := os.Lstat(expFile)
	assert.NoError(t, err)
	assert.Equal(t, int64(52), fi.Size())
}

type mockExporter struct{}

func (mock *mockExporter) Export(w io.Writer, _ string) (int, error) {
	_, err := w.Write([]byte("some export blah blah 1234567890"))
	return 1000, err
}
