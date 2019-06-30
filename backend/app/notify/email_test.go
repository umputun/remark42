package notify

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEmail(t *testing.T) {
	email, err := NewEmail("", 0, "", "", 0)
	assert.EqualError(t, err, "[WARN] error connecting to '':0 with username '': dial tcp :0: connect: can't assign requested address")
	assert.NotNil(t, email, "despite the error we got object reference")
	assert.Equal(t, email.server, "")
	assert.Equal(t, email.port, 0)
	assert.Equal(t, email.username, "")
	assert.Equal(t, email.password, "")
	assert.Equal(t, email.keepAlive, 30*time.Second, "default value if keepAlive is not defined is set to 30s")
	assert.Equal(t, "email: ''@'':0", email.String())
}
