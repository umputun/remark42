package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-pkgz/auth/v2/avatar"
	"github.com/jessevdk/go-flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAvatar_Execute(t *testing.T) {
	defer os.RemoveAll("/tmp/ava-test")

	// from fs to bolt
	cmd := AvatarCommand{migrate: (&avatarMigratorMock{retCount: 100}).Migrate}
	cmd.SetCommon(CommonOpts{RemarkURL: "", SharedSecret: "123456"})
	p := flags.NewParser(&cmd, flags.Default)
	_, err := p.ParseArgs([]string{"--src.type=fs", "--src.fs.path=/tmp/ava-test", "--dst.type=bolt",
		"--dst.bolt.file=/tmp/ava-test.db"})
	require.NoError(t, err)
	defer os.Remove("/tmp/ava-test.db")
	err = cmd.Execute(nil)
	assert.NoError(t, err)

	// failed
	cmd = AvatarCommand{migrate: (&avatarMigratorMock{retCount: 0, retError: fmt.Errorf("failed blah")}).Migrate}
	cmd.SetCommon(CommonOpts{RemarkURL: "", SharedSecret: "123456"})
	p = flags.NewParser(&cmd, flags.Default)
	_, err = p.ParseArgs([]string{"--src.type=fs", "--src.fs.path=/tmp/ava-test", "--dst.type=bolt",
		"--dst.bolt.file=/tmp/ava-test2.db"})
	require.NoError(t, err)
	defer os.Remove("/tmp/ava-test2.db")
	err = cmd.Execute(nil)
	assert.Error(t, err, "failed blah")
}

type avatarMigratorMock struct {
	called   int
	retError error
	retCount int
}

func (a *avatarMigratorMock) Migrate(_, _ avatar.Store) (int, error) {
	a.called++
	return a.retCount, a.retError
}

func TestAvatar_ExecuteURIStore(t *testing.T) {
	tmp := t.TempDir()
	mock := &avatarMigratorMock{retCount: 1}
	cmd := AvatarCommand{migrate: mock.Migrate}
	cmd.SetCommon(CommonOpts{RemarkURL: "", SharedSecret: "123456"})
	p := flags.NewParser(&cmd, flags.Default)
	_, err := p.ParseArgs([]string{"--src.type=uri", "--src.uri=" + filepath.Join(tmp, "src"),
		"--dst.type=uri", "--dst.uri=bolt://" + filepath.Join(tmp, "dst.db")})
	require.NoError(t, err)
	require.NoError(t, cmd.Execute(nil), "the uri store type offered by --src.type and --dst.type is accepted")
	assert.Equal(t, 1, mock.called)
}
