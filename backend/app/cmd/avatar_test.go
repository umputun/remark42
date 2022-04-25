package cmd

import (
	"fmt"
	"os"
	"testing"

	"github.com/go-pkgz/auth/avatar"
	"github.com/jessevdk/go-flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAvatar_Execute(t *testing.T) {
	defer os.RemoveAll("/tmp/ava-test")

	// from fs to bolt
	cmd := AvatarCommand{migrator: &avatarMigratorMock{retCount: 100}}
	cmd.SetCommon(CommonOpts{RemarkURL: "", SharedSecret: "123456"})
	p := flags.NewParser(&cmd, flags.Default)
	_, err := p.ParseArgs([]string{"--src.type=fs", "--src.fs.path=/tmp/ava-test", "--dst.type=bolt",
		"--dst.bolt.file=/tmp/ava-test.db"})
	require.NoError(t, err)
	err = cmd.Execute(nil)
	assert.NoError(t, err)

	// failed
	cmd = AvatarCommand{migrator: &avatarMigratorMock{retCount: 0, retError: fmt.Errorf("failed blah")}}
	cmd.SetCommon(CommonOpts{RemarkURL: "", SharedSecret: "123456"})
	p = flags.NewParser(&cmd, flags.Default)
	_, err = p.ParseArgs([]string{"--src.type=fs", "--src.fs.path=/tmp/ava-test", "--dst.type=bolt",
		"--dst.bolt.file=/tmp/ava-test2.db"})
	require.NoError(t, err)
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
