package cmd

import (
	"errors"
	"os"
	"testing"

	"github.com/go-pkgz/auth/avatar"
	flags "github.com/jessevdk/go-flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAvatar_Execute(t *testing.T) {

	mongoURL := os.Getenv("MONGO_TEST")
	if mongoURL == "" {
		mongoURL = "mongodb://localhost:27017/test"
	}
	if mongoURL == "skip" {
		t.Skip("skip mongo app test")
	}
	defer os.RemoveAll("/tmp/ava-test")

	// from fs to mongo
	cmd := AvatarCommand{migrator: &avatarMigratorMock{retCount: 100}}
	cmd.SetCommon(CommonOpts{RemarkURL: "", SharedSecret: "123456"})
	p := flags.NewParser(&cmd, flags.Default)
	_, err := p.ParseArgs([]string{"--src.type=fs", "--src.fs.path=/tmp/ava-test", "--dst.type=mongo",
		"--mongo.url=" + mongoURL, "--mongo.db=test_remark"})
	require.Nil(t, err)
	err = cmd.Execute(nil)
	assert.NoError(t, err)

	// from fs to bolt
	cmd = AvatarCommand{migrator: &avatarMigratorMock{retCount: 100}}
	cmd.SetCommon(CommonOpts{RemarkURL: "", SharedSecret: "123456"})
	p = flags.NewParser(&cmd, flags.Default)
	_, err = p.ParseArgs([]string{"--src.type=fs", "--src.fs.path=/tmp/ava-test", "--dst.type=bolt",
		"--dst.bolt.file=/tmp/ava-test.db"})
	require.Nil(t, err)
	err = cmd.Execute(nil)
	assert.NoError(t, err)

	// failed
	cmd = AvatarCommand{migrator: &avatarMigratorMock{retCount: 0, retError: errors.New("failed blah")}}
	cmd.SetCommon(CommonOpts{RemarkURL: "", SharedSecret: "123456"})
	p = flags.NewParser(&cmd, flags.Default)
	_, err = p.ParseArgs([]string{"--src.type=fs", "--src.fs.path=/tmp/ava-test", "--dst.type=mongo",
		"--mongo.url=" + mongoURL, "--mongo.db=test_remark"})
	require.Nil(t, err)
	err = cmd.Execute(nil)
	assert.Error(t, err, "failed blah")
}

type avatarMigratorMock struct {
	called   int
	retError error
	retCount int
}

func (a *avatarMigratorMock) Migrate(dst, src avatar.Store) (int, error) {
	a.called++
	return a.retCount, a.retError
}
