/*
 * // Copyleft (ɔ) 2018 The Caliopen contributors.
 * // Use of this source code is governed by a GNU AFFERO GENERAL PUBLIC
 * // license (AGPL) that can be found in the LICENSE file.
 */

package backends

import (
	. "github.com/CaliOpen/Caliopen/src/backend/defs/go-objects"
)

type (
	IdentityStorage interface {
		CreateRemoteIdentity(rId *RemoteIdentity) error
		RetrieveRemoteIdentity(userId, identifier string) (*RemoteIdentity, error)
		UpdateRemoteIdentity(rId *RemoteIdentity, fields map[string]interface{}) error
		GetLocalsIdentities(user_id string) (identities []LocalIdentity, err error)
		RetrieveAllRemotes() (<-chan *RemoteIdentity, error)
		Close()
	}
)
