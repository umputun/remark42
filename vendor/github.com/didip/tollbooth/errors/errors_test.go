package errors

import "testing"

func TestError(t *testing.T) {
	errs := HTTPError{"blah", 429}
	if errs.Error() == "" {
		t.Errorf("Unable to print Error(). Value: %v", errs.Error())
	}
}
