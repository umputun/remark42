package render

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/ajg/form"
)

// Decode is a package-level variable set to our default Decoder. We do this
// because it allows you to set render.Decode to another function with the
// same function signature, while also utilizing the render.Decoder() function
// itself. Effectively, allowing you to easily add your own logic to the package
// defaults. For example, maybe you want to impose a limit on the number of
// bytes allowed to be read from the request body.
var Decode = DefaultDecoder

// DefaultDecoder detects the correct decoder for use on an HTTP request and
// marshals into a given interface.
func DefaultDecoder(r *http.Request, v interface{}) error {
	var err error

	switch GetRequestContentType(r) {
	case ContentTypeJSON:
		err = DecodeJSON(r.Body, v)
	case ContentTypeXML:
		err = DecodeXML(r.Body, v)
	case ContentTypeForm:
		err = DecodeForm(r.Body, v)
	default:
		err = errors.New("render: unable to automatically decode the request content type")
	}

	return err
}

// DecodeJSON decodes a given reader into an interface using the json decoder.
func DecodeJSON(r io.Reader, v interface{}) error {
	defer io.Copy(ioutil.Discard, r) //nolint:errcheck
	return json.NewDecoder(r).Decode(v)
}

// DecodeXML decodes a given reader into an interface using the xml decoder.
func DecodeXML(r io.Reader, v interface{}) error {
	defer io.Copy(ioutil.Discard, r) //nolint:errcheck
	return xml.NewDecoder(r).Decode(v)
}

// DecodeForm decodes a given reader into an interface using the form decoder.
func DecodeForm(r io.Reader, v interface{}) error {
	decoder := form.NewDecoder(r) //nolint:errcheck
	return decoder.Decode(v)
}
