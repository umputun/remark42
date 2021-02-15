/*
 * Copyright 2019 Umputun. All rights reserved.
 * Use of this source code is governed by a MIT-style
 * license that can be found in the LICENSE file.
 */

package accessor

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/umputun/remark42/backend/app/store/admin"
)

func TestMemAdmin_Get(t *testing.T) {

	adm := NewMemAdminStore("secret")
	var ms admin.Store = adm

	adm.data = map[string]AdminRec{
		"site1": {"site1", []string{"i11", "i12"}, "e1", true, 0},
	}
	adm.Set("site2", AdminRec{"site2", []string{"i21", "i22"}, "e2", true, 0})
	adm.Set("site3", AdminRec{"site3", []string{"i21", "i22"}, "e3", false, 0})

	admins, err := ms.Admins("site1")
	assert.NoError(t, err)
	assert.Equal(t, []string{"i11", "i12"}, admins)
	email, err := ms.Email("site1")
	assert.NoError(t, err)
	assert.Equal(t, "e1", email)
	key, err := ms.Key("any")
	assert.NoError(t, err)
	assert.Equal(t, "secret", key)

	admins, err = ms.Admins("site2")
	assert.NoError(t, err)
	assert.Equal(t, []string{"i21", "i22"}, admins)
	email, err = ms.Email("site2")
	assert.NoError(t, err)
	assert.Equal(t, "e2", email)
	key, err = ms.Key("any")
	assert.NoError(t, err)
	assert.Equal(t, "secret", key)

	admins, err = ms.Admins("no-site-in-db")
	assert.EqualError(t, err, "site no-site-in-db not found")
	assert.Empty(t, admins)

	email, err = ms.Email("no-site-in-db")
	assert.EqualError(t, err, "site no-site-in-db not found")
	assert.Empty(t, email)

	enabled, err := ms.Enabled("site1")
	assert.NoError(t, err)
	assert.True(t, enabled)

	enabled, err = ms.Enabled("site3")
	assert.NoError(t, err)
	assert.False(t, enabled)

	enabled, err = ms.Enabled("no-site-in-db")
	assert.EqualError(t, err, "site no-site-in-db not found")
	assert.False(t, enabled)

	err = ms.OnEvent("site1", admin.EvCreate)
	assert.NoError(t, err)

	err = ms.OnEvent("no-site-in-db", admin.EvCreate)
	assert.Error(t, err)
}
