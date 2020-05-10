package templates

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewFS(t *testing.T) {
	fs := NewFS()
	assert.NotNil(t, &fs)
}

func TestFS_ReadFile(t *testing.T) {
	fs := NewFS()

	file, err := fs.ReadFile("testdata/template.html.tmpl")
	assert.NoError(t, err)
	assert.Equal(t, []byte("template\n"), file)

	file, err = fs.ReadFile("testdata/bad_path.html.tmpl")
	assert.Error(t, err)
	assert.Nil(t, file)
}
