package templates

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Read(t *testing.T) {
	file, err := Read("testdata/template.html.tmpl")
	assert.NoError(t, err)
	assert.Equal(t, []byte("template\n"), file)

	file, err = Read("testdata/bad_path.html.tmpl")
	assert.Error(t, err)
	assert.Nil(t, file)
}
