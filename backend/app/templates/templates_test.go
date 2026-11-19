package templates

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Read(t *testing.T) {
	file, err := Read("testdata/template.html.tmpl")
	assert.NoError(t, err)
	assert.Equal(t, []byte("template\n"), file)

	file, err = Read("testdata/bad_path.html.tmpl")
	assert.Error(t, err)
	assert.Nil(t, file)

	// a name with no file on disk comes from the embedded templates
	file, err = Read("error_response.html.tmpl")
	require.NoError(t, err)
	want, err := os.ReadFile("static/error_response.html.tmpl")
	require.NoError(t, err)
	assert.Equal(t, want, file)
}
