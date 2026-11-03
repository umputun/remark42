package lgr

import (
	"bytes"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
)

// secretList is an immutable list of values to mask, sorted longest first
type secretList [][]byte

// hide masks the values of both lists in data. Values are taken longest first across the two lists,
// so a value containing a shorter one is replaced as a whole, whichever list holds it.
func (sl secretList) hide(data []byte, other secretList) []byte {
	for len(sl) > 0 || len(other) > 0 {
		var v []byte
		if len(other) == 0 || (len(sl) > 0 && secretOrder(sl[0], other[0]) <= 0) {
			v, sl = sl[0], sl[1:]
		} else {
			v, other = other[0], other[1:]
		}
		data = bytes.ReplaceAll(data, v, secretReplacement)
	}
	return data
}

// secretSet is a set of values to mask, changeable while other goroutines read it.
// Writers replace the published list and never modify it, so readers need no lock.
type secretSet struct {
	lock sync.Mutex
	vals atomic.Pointer[secretList]
}

// list returns the published values, safe to call on a nil set
func (s *secretSet) list() secretList {
	if s == nil {
		return nil
	}
	if p := s.vals.Load(); p != nil {
		return *p
	}
	return nil
}

func (s *secretSet) add(vals ...string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	res := slices.Clone(s.list())
	for _, v := range vals {
		if strings.TrimSpace(v) == "" {
			continue
		}
		if slices.ContainsFunc(res, func(b []byte) bool { return string(b) == v }) {
			continue
		}
		res = append(res, []byte(v))
	}
	slices.SortFunc(res, secretOrder)
	s.vals.Store(&res)
}

func (s *secretSet) remove(vals ...string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	res := slices.DeleteFunc(slices.Clone(s.list()), func(b []byte) bool { return slices.Contains(vals, string(b)) })
	s.vals.Store(&res)
}

func secretOrder(a, b []byte) int {
	if len(a) != len(b) {
		return len(b) - len(a)
	}
	return bytes.Compare(a, b)
}
