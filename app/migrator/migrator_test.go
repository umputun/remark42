package migrator

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMigrator_RemoveOldBackupFiles(t *testing.T) {
	loc := "tmp/remark-backups.test"
	defer os.RemoveAll(loc)

	os.MkdirAll(loc, 0700)
	for i := 0; i < 10; i++ {
		fname := fmt.Sprintf("%s/backup-site1-201712%02d.gz", loc, i)
		err := ioutil.WriteFile(fname, []byte("blah"), 0600)
		assert.Nil(t, err)
	}

	removeOldBackupFiles(loc, "site1", 3)
	ff, err := ioutil.ReadDir(loc)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(ff), "should keep 3 files only")
	assert.Equal(t, "backup-site1-20171207.gz", ff[0].Name())
	assert.Equal(t, "backup-site1-20171208.gz", ff[1].Name())
	assert.Equal(t, "backup-site1-20171209.gz", ff[2].Name())
}
