package migrator

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackup_RemoveOldBackupFiles(t *testing.T) {
	loc := t.TempDir()

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

func TestBackup_RemoveOldBackupFilesOfSiteOnly(t *testing.T) {
	loc := t.TempDir()
	for _, site := range []string{"radio", "radio-t"} {
		for i := 1; i <= 5; i++ {
			fname := fmt.Sprintf("%s/backup-%s-201712%02d.gz", loc, site, i)
			require.NoError(t, os.WriteFile(fname, []byte("blah"), 0o600))
		}
	}

	bk := AutoBackup{BackupLocation: loc, SiteID: "radio", KeepMax: 3}
	bk.removeOldBackupFiles()

	ff, err := os.ReadDir(loc)
	require.NoError(t, err)
	names := make([]string, 0, len(ff))
	for _, f := range ff {
		names = append(names, f.Name())
	}
	assert.Equal(t, []string{
		"backup-radio-20171203.gz", "backup-radio-20171204.gz", "backup-radio-20171205.gz",
		"backup-radio-t-20171201.gz", "backup-radio-t-20171202.gz", "backup-radio-t-20171203.gz",
		"backup-radio-t-20171204.gz", "backup-radio-t-20171205.gz",
	}, names, "pruning radio keeps every backup of radio-t")
}

func TestBackup_MakeBackupFailedExportKeepsEarlierBackup(t *testing.T) {
	loc := t.TempDir()
	synctest.Test(t, func(t *testing.T) { // the fake clock keeps both backups on the same day
		good := AutoBackup{BackupLocation: loc, SiteID: "site1", KeepMax: 3, Exporter: &mockExporter{}}
		fname, err := good.makeBackup(t.Context(), time.Now())
		require.NoError(t, err)

		bad := AutoBackup{BackupLocation: loc, SiteID: "site1", KeepMax: 3, Exporter: &failingExporter{}}
		_, err = bad.makeBackup(t.Context(), time.Now())
		require.Error(t, err)

		assert.Equal(t, exportedPayload, gzContent(t, fname), "a failed export leaves the earlier backup intact")
		ff, err := os.ReadDir(loc)
		require.NoError(t, err)
		require.Len(t, ff, 1, "a failed export leaves no file behind")
		assert.Equal(t, filepath.Base(fname), ff[0].Name())
	})
}

func TestBackup_MakeBackup(t *testing.T) {
	loc := t.TempDir()

	bk := AutoBackup{BackupLocation: loc, SiteID: "site1", KeepMax: 3, Exporter: &mockExporter{}}
	day := time.Date(2017, 12, 25, 10, 0, 0, 0, time.Local)
	fname, err := bk.makeBackup(t.Context(), day)
	assert.NoError(t, err)
	expFile := loc + "/backup-site1-20171225.gz"
	assert.Equal(t, expFile, fname)

	assert.Equal(t, exportedPayload, gzContent(t, expFile))
}

func TestBackup_MakeBackupStopsWhenContextDone(t *testing.T) {
	loc := t.TempDir()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	bk := AutoBackup{BackupLocation: loc, SiteID: "site1", KeepMax: 3, Exporter: &mockExporter{}}
	_, err := bk.makeBackup(ctx, time.Now())
	require.ErrorIs(t, err, context.Canceled)
	ff, err := os.ReadDir(loc)
	require.NoError(t, err)
	assert.Empty(t, ff, "a stopped export leaves no file behind")
}

func TestBackup_DoWritesBackupAtStart(t *testing.T) {
	loc := t.TempDir()
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		today, yesterday := start.Format("20060102"), start.AddDate(0, 0, -1).Format("20060102")
		// none of these is the backup of site1 for today
		for _, name := range []string{"backup-site1-" + yesterday + ".gz", "backup-site1-t-" + today + ".gz",
			"backup-site1-" + today + ".gz.tmp", "userbackup-site1-" + today + "T101010.gz"} {
			require.NoError(t, os.WriteFile(filepath.Join(loc, name), []byte("other"), 0o600))
		}
		exporter := &mockExporter{}
		bk := AutoBackup{BackupLocation: loc, SiteID: "site1", KeepMax: 3, Exporter: exporter, Duration: 24 * time.Hour}
		stop := runDo(t, bk)

		synctest.Wait() // Do is blocked on its ticker: the start backup is done, and no time has passed
		assert.Equal(t, start, time.Now())
		assert.Equal(t, int32(1), exporter.calls.Load(), "a start with no backup for today makes one")
		assert.Equal(t, exportedPayload, gzContent(t, filepath.Join(loc, "backup-site1-"+today+".gz")))
		exp := []string{"backup-site1-" + yesterday + ".gz", "backup-site1-" + today + ".gz",
			"backup-site1-t-" + today + ".gz", "userbackup-site1-" + today + "T101010.gz"}
		slices.Sort(exp) // whether yesterday sorts first depends on the local time zone at the fake clock's start
		assert.Equal(t, exp, dirNames(t, loc), "the other files are kept, the temporary one is replaced")
		stop()
	})
}

func TestBackup_DoSkipsStartBackupWhenTodayHasOne(t *testing.T) {
	loc := t.TempDir()
	synctest.Test(t, func(t *testing.T) {
		fname := filepath.Join(loc, "backup-site1-"+time.Now().Format("20060102")+".gz")
		require.NoError(t, os.WriteFile(fname, []byte("earlier"), 0o600))
		exp := &mockExporter{}
		bk := AutoBackup{BackupLocation: loc, SiteID: "site1", KeepMax: 3, Exporter: exp, Duration: 24 * time.Hour}
		stop := runDo(t, bk)

		synctest.Wait()
		assert.Equal(t, int32(0), exp.calls.Load(), "a start on a day with a backup makes no other")
		data, err := os.ReadFile(fname) //nolint:gosec // path is built by the test
		require.NoError(t, err)
		assert.Equal(t, "earlier", string(data), "the backup of the day is left as it was")
		stop()
	})
}

func TestBackup_DoWritesBackupEveryDuration(t *testing.T) {
	loc := t.TempDir()
	synctest.Test(t, func(t *testing.T) {
		exp := &mockExporter{}
		bk := AutoBackup{BackupLocation: loc, SiteID: "site1", KeepMax: 2, Exporter: exp, Duration: 24 * time.Hour}
		stop := runDo(t, bk)
		day0 := time.Now().Format("20060102")

		synctest.Wait()
		require.Equal(t, int32(1), exp.calls.Load(), "the start backup")

		time.Sleep(24 * time.Hour)
		synctest.Wait()
		day1 := time.Now().Format("20060102")
		assert.Equal(t, int32(2), exp.calls.Load(), "the backup a day after start")
		assert.Equal(t, exportedPayload, gzContent(t, filepath.Join(loc, "backup-site1-"+day1+".gz")))

		time.Sleep(24 * time.Hour)
		synctest.Wait()
		day2 := time.Now().Format("20060102")
		assert.Equal(t, int32(3), exp.calls.Load(), "the backup two days after start")
		assert.Equal(t, []string{"backup-site1-" + day1 + ".gz", "backup-site1-" + day2 + ".gz"}, dirNames(t, loc),
			"the backup of %s is pruned to keep KeepMax", day0)
		stop()
	})
}

func TestBackup_DoPrunesAfterStartBackup(t *testing.T) {
	loc := t.TempDir()
	synctest.Test(t, func(t *testing.T) {
		now := time.Now()
		for i := 1; i <= 4; i++ {
			name := "backup-site1-" + now.AddDate(0, 0, -i).Format("20060102") + ".gz"
			require.NoError(t, os.WriteFile(filepath.Join(loc, name), []byte("old"), 0o600))
		}
		bk := AutoBackup{BackupLocation: loc, SiteID: "site1", KeepMax: 3, Exporter: &mockExporter{}, Duration: 24 * time.Hour}
		stop := runDo(t, bk)

		synctest.Wait()
		assert.Equal(t, []string{"backup-site1-" + now.AddDate(0, 0, -2).Format("20060102") + ".gz",
			"backup-site1-" + now.AddDate(0, 0, -1).Format("20060102") + ".gz",
			"backup-site1-" + now.Format("20060102") + ".gz"}, dirNames(t, loc), "the start backup and the two newest before it")
		stop()
	})
}

func TestBackup_DoKeepsBackupsWhenStartExportIsEmpty(t *testing.T) {
	// a server started on an empty store, as before a restore, must not push its recovery points out
	tbl := []struct {
		name     string
		keepMax  int
		exporter Exporter
		today    bool // whether the backup of today is expected
	}{
		{"empty export, the only backup would be replaced", 1, &emptyExporter{}, false},
		{"empty export, room for one more", 2, &emptyExporter{}, true},
		{"export with comments replaces the oldest", 1, &mockExporter{}, true},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			loc := t.TempDir()
			synctest.Test(t, func(t *testing.T) {
				now := time.Now()
				yesterday := "backup-site1-" + now.AddDate(0, 0, -1).Format("20060102") + ".gz"
				today := "backup-site1-" + now.Format("20060102") + ".gz"
				require.NoError(t, os.WriteFile(filepath.Join(loc, yesterday), []byte("recovery point"), 0o600))
				bk := AutoBackup{BackupLocation: loc, SiteID: "site1", KeepMax: tt.keepMax, Exporter: tt.exporter, Duration: 24 * time.Hour}
				stop := runDo(t, bk)

				synctest.Wait()
				switch {
				case !tt.today:
					assert.Equal(t, []string{yesterday}, dirNames(t, loc), "the empty export is dropped, no temporary file is left")
					data, err := os.ReadFile(filepath.Join(loc, yesterday)) //nolint:gosec // test file
					require.NoError(t, err)
					assert.Equal(t, "recovery point", string(data))
				case tt.keepMax == 1:
					assert.Equal(t, []string{today}, dirNames(t, loc))
				default:
					assert.Equal(t, []string{yesterday, today}, dirNames(t, loc))
				}
				stop()
			})
		})
	}
}

func TestBackup_DoRetriesFailedStartBackupOnSchedule(t *testing.T) {
	loc := t.TempDir()
	synctest.Test(t, func(t *testing.T) {
		exp := &failOnceExporter{}
		bk := AutoBackup{BackupLocation: loc, SiteID: "site1", KeepMax: 3, Exporter: exp, Duration: 24 * time.Hour}
		stop := runDo(t, bk)

		synctest.Wait()
		require.Equal(t, int32(1), exp.calls.Load(), "the start backup was tried")
		assert.Empty(t, dirNames(t, loc), "a failed start backup leaves no file behind")

		time.Sleep(24 * time.Hour)
		synctest.Wait()
		assert.Equal(t, int32(2), exp.calls.Load())
		assert.Equal(t, []string{"backup-site1-" + time.Now().Format("20060102") + ".gz"}, dirNames(t, loc),
			"the scheduled backup after a failed start one")
		stop()
	})
}

// runDo runs bk.Do in the background; the returned function stops it and waits for it to return
func runDo(t *testing.T, bk AutoBackup) (stop func()) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		bk.Do(ctx)
		close(done)
	}()
	return func() {
		cancel()
		<-done
	}
}

func dirNames(t *testing.T, dir string) []string {
	t.Helper()
	ff, err := os.ReadDir(dir)
	require.NoError(t, err)
	names := make([]string, 0, len(ff))
	for _, f := range ff {
		names = append(names, f.Name())
	}
	return names
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

type mockExporter struct{ calls atomic.Int32 }

func (mock *mockExporter) Export(w io.Writer, _ string) (int, error) {
	mock.calls.Add(1)
	_, err := w.Write([]byte(exportedPayload))
	return 1000, err
}

// emptyExporter exports a site with no comments
type emptyExporter struct{}

func (*emptyExporter) Export(w io.Writer, _ string) (int, error) {
	_, err := w.Write([]byte(`{"version":1,"users":[],"posts":[]}`))
	return 0, err
}

// failOnceExporter fails the first export and writes the payload on the later ones
type failOnceExporter struct{ calls atomic.Int32 }

func (e *failOnceExporter) Export(w io.Writer, _ string) (int, error) {
	if e.calls.Add(1) == 1 {
		return 0, errors.New("export failed")
	}
	_, err := w.Write([]byte(exportedPayload))
	return 1000, err
}

// failingExporter writes part of an export and then fails
type failingExporter struct{}

func (*failingExporter) Export(w io.Writer, _ string) (int, error) {
	if _, err := w.Write([]byte("partial")); err != nil {
		return 0, err
	}
	return 0, errors.New("export failed")
}
