package admin

// Store defines interface returning admins info for given site
type Store interface {
	Admins(siteID string) (ids []string)
	Email(siteID string) (email string)
}

// StaticStore implements keys.Store with a single, predefined key
type StaticStore struct {
	admins []string
	email  string
}

// NewStaticStore makes StaticStore instance with given key
func NewStaticStore(admins []string, email string) *StaticStore {
	return &StaticStore{admins: admins, email: email}
}

// Admins returns static list of admin's ids, the same for all sites
func (s *StaticStore) Admins(string) (ids []string) {
	return s.admins
}

// Email gets static email address
func (s *StaticStore) Email(string) (email string) {
	return s.email
}
