// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package admin

import (
	"sync"
)

// Ensure, that StoreMock does implement Store.
// If this is not the case, regenerate this file with moq.
var _ Store = &StoreMock{}

// StoreMock is a mock implementation of Store.
//
//	func TestSomethingThatUsesStore(t *testing.T) {
//
//		// make and configure a mocked Store
//		mockedStore := &StoreMock{
//			AdminsFunc: func(siteID string) ([]string, error) {
//				panic("mock out the Admins method")
//			},
//			EmailFunc: func(siteID string) (string, error) {
//				panic("mock out the Email method")
//			},
//			EnabledFunc: func(siteID string) (bool, error) {
//				panic("mock out the Enabled method")
//			},
//			KeyFunc: func(siteID string) (string, error) {
//				panic("mock out the Key method")
//			},
//			OnEventFunc: func(siteID string, et EventType) error {
//				panic("mock out the OnEvent method")
//			},
//		}
//
//		// use mockedStore in code that requires Store
//		// and then make assertions.
//
//	}
type StoreMock struct {
	// AdminsFunc mocks the Admins method.
	AdminsFunc func(siteID string) ([]string, error)

	// EmailFunc mocks the Email method.
	EmailFunc func(siteID string) (string, error)

	// EnabledFunc mocks the Enabled method.
	EnabledFunc func(siteID string) (bool, error)

	// KeyFunc mocks the Key method.
	KeyFunc func(siteID string) (string, error)

	// OnEventFunc mocks the OnEvent method.
	OnEventFunc func(siteID string, et EventType) error

	// calls tracks calls to the methods.
	calls struct {
		// Admins holds details about calls to the Admins method.
		Admins []struct {
			// SiteID is the siteID argument value.
			SiteID string
		}
		// Email holds details about calls to the Email method.
		Email []struct {
			// SiteID is the siteID argument value.
			SiteID string
		}
		// Enabled holds details about calls to the Enabled method.
		Enabled []struct {
			// SiteID is the siteID argument value.
			SiteID string
		}
		// Key holds details about calls to the Key method.
		Key []struct {
			// SiteID is the siteID argument value.
			SiteID string
		}
		// OnEvent holds details about calls to the OnEvent method.
		OnEvent []struct {
			// SiteID is the siteID argument value.
			SiteID string
			// Et is the et argument value.
			Et EventType
		}
	}
	lockAdmins  sync.RWMutex
	lockEmail   sync.RWMutex
	lockEnabled sync.RWMutex
	lockKey     sync.RWMutex
	lockOnEvent sync.RWMutex
}

// Admins calls AdminsFunc.
func (mock *StoreMock) Admins(siteID string) ([]string, error) {
	if mock.AdminsFunc == nil {
		panic("StoreMock.AdminsFunc: method is nil but Store.Admins was just called")
	}
	callInfo := struct {
		SiteID string
	}{
		SiteID: siteID,
	}
	mock.lockAdmins.Lock()
	mock.calls.Admins = append(mock.calls.Admins, callInfo)
	mock.lockAdmins.Unlock()
	return mock.AdminsFunc(siteID)
}

// AdminsCalls gets all the calls that were made to Admins.
// Check the length with:
//
//	len(mockedStore.AdminsCalls())
func (mock *StoreMock) AdminsCalls() []struct {
	SiteID string
} {
	var calls []struct {
		SiteID string
	}
	mock.lockAdmins.RLock()
	calls = mock.calls.Admins
	mock.lockAdmins.RUnlock()
	return calls
}

// Email calls EmailFunc.
func (mock *StoreMock) Email(siteID string) (string, error) {
	if mock.EmailFunc == nil {
		panic("StoreMock.EmailFunc: method is nil but Store.Email was just called")
	}
	callInfo := struct {
		SiteID string
	}{
		SiteID: siteID,
	}
	mock.lockEmail.Lock()
	mock.calls.Email = append(mock.calls.Email, callInfo)
	mock.lockEmail.Unlock()
	return mock.EmailFunc(siteID)
}

// EmailCalls gets all the calls that were made to Email.
// Check the length with:
//
//	len(mockedStore.EmailCalls())
func (mock *StoreMock) EmailCalls() []struct {
	SiteID string
} {
	var calls []struct {
		SiteID string
	}
	mock.lockEmail.RLock()
	calls = mock.calls.Email
	mock.lockEmail.RUnlock()
	return calls
}

// Enabled calls EnabledFunc.
func (mock *StoreMock) Enabled(siteID string) (bool, error) {
	if mock.EnabledFunc == nil {
		panic("StoreMock.EnabledFunc: method is nil but Store.Enabled was just called")
	}
	callInfo := struct {
		SiteID string
	}{
		SiteID: siteID,
	}
	mock.lockEnabled.Lock()
	mock.calls.Enabled = append(mock.calls.Enabled, callInfo)
	mock.lockEnabled.Unlock()
	return mock.EnabledFunc(siteID)
}

// EnabledCalls gets all the calls that were made to Enabled.
// Check the length with:
//
//	len(mockedStore.EnabledCalls())
func (mock *StoreMock) EnabledCalls() []struct {
	SiteID string
} {
	var calls []struct {
		SiteID string
	}
	mock.lockEnabled.RLock()
	calls = mock.calls.Enabled
	mock.lockEnabled.RUnlock()
	return calls
}

// Key calls KeyFunc.
func (mock *StoreMock) Key(siteID string) (string, error) {
	if mock.KeyFunc == nil {
		panic("StoreMock.KeyFunc: method is nil but Store.Key was just called")
	}
	callInfo := struct {
		SiteID string
	}{
		SiteID: siteID,
	}
	mock.lockKey.Lock()
	mock.calls.Key = append(mock.calls.Key, callInfo)
	mock.lockKey.Unlock()
	return mock.KeyFunc(siteID)
}

// KeyCalls gets all the calls that were made to Key.
// Check the length with:
//
//	len(mockedStore.KeyCalls())
func (mock *StoreMock) KeyCalls() []struct {
	SiteID string
} {
	var calls []struct {
		SiteID string
	}
	mock.lockKey.RLock()
	calls = mock.calls.Key
	mock.lockKey.RUnlock()
	return calls
}

// OnEvent calls OnEventFunc.
func (mock *StoreMock) OnEvent(siteID string, et EventType) error {
	if mock.OnEventFunc == nil {
		panic("StoreMock.OnEventFunc: method is nil but Store.OnEvent was just called")
	}
	callInfo := struct {
		SiteID string
		Et     EventType
	}{
		SiteID: siteID,
		Et:     et,
	}
	mock.lockOnEvent.Lock()
	mock.calls.OnEvent = append(mock.calls.OnEvent, callInfo)
	mock.lockOnEvent.Unlock()
	return mock.OnEventFunc(siteID, et)
}

// OnEventCalls gets all the calls that were made to OnEvent.
// Check the length with:
//
//	len(mockedStore.OnEventCalls())
func (mock *StoreMock) OnEventCalls() []struct {
	SiteID string
	Et     EventType
} {
	var calls []struct {
		SiteID string
		Et     EventType
	}
	mock.lockOnEvent.RLock()
	calls = mock.calls.OnEvent
	mock.lockOnEvent.RUnlock()
	return calls
}
