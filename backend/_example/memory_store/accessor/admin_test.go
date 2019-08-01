/*
 * Copyright 2019 Umputun. All rights reserved.
 * Use of this source code is governed by a MIT-style
 * license that can be found in the LICENSE file.
 */

package accessor

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/umputun/remark/backend/app/store/admin"
)

func TestMemAdmin_Get(t *testing.T) {

	adm := NewMemAdminStore("secret")
	var ms admin.Store = adm

	adm.data = map[string]AdminRec{
		"site1": {"site1", []string{"i11", "i12"}, "e1"},
		"site2": {"site2", []string{"i21", "i22"}, "e2"},
	}

	admins, err := ms.Admins("site1")
	assert.NoError(t, err)
	assert.Equal(t, []string{"i11", "i12"}, admins)
	email, err := ms.Email("site1")
	assert.NoError(t, err)
	assert.Equal(t, "e1", email)
	key, err := ms.Key()
	assert.NoError(t, err)
	assert.Equal(t, "secret", key)

	admins, err = ms.Admins("site2")
	assert.NoError(t, err)
	assert.Equal(t, []string{"i21", "i22"}, admins)
	email, err = ms.Email("site2")
	assert.NoError(t, err)
	assert.Equal(t, "e2", email)
	key, err = ms.Key()
	assert.NoError(t, err)
	assert.Equal(t, "secret", key)

	admins, err = ms.Admins("no-site-in-db")
	assert.EqualError(t, err, "site no-site-in-db not found")

	email, err = ms.Email("no-site-in-db")
	assert.EqualError(t, err, "site no-site-in-db not found")
}
