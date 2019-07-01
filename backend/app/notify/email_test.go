package notify

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEmail(t *testing.T) {
	email, err := NewEmail(EmailParams{})
	assert.Error(t, err, "No connection established with empty address and port zero")
	assert.NotNil(t, email, "despite the error we got object reference")
	assert.Equal(t, email.server, "")
	assert.Equal(t, email.port, 0)
	assert.Equal(t, email.username, "")
	assert.Equal(t, email.password, "")
	assert.Equal(t, email.keepAlive, 30*time.Second, "default value if keepAlive is not defined is set to 30s")
	assert.Equal(t, "email: ''@'':0", email.String())
}
