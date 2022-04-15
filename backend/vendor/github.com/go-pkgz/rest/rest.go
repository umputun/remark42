// Package rest provides common middlewares and helpers for rest services
package rest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

// JSON is a map alias, just for convenience
type JSON map[string]interface{}

// RenderJSON sends data as json
func RenderJSON(w http.ResponseWriter, data interface{}) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(true)
	if err := enc.Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(buf.Bytes())
}

// RenderJSONFromBytes sends binary data as json
func RenderJSONFromBytes(w http.ResponseWriter, r *http.Request, data []byte) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if _, err := w.Write(data); err != nil {
		return errors.Wrapf(err, "failed to send response to %s", r.RemoteAddr)
	}
	return nil
}

// RenderJSONWithHTML allows html tags and forces charset=utf-8
func RenderJSONWithHTML(w http.ResponseWriter, r *http.Request, v interface{}) error {

	encodeJSONWithHTML := func(v interface{}) ([]byte, error) {
		buf := &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(v); err != nil {
			return nil, errors.Wrap(err, "json encoding failed")
		}
		return buf.Bytes(), nil
	}

	data, err := encodeJSONWithHTML(v)
	if err != nil {
		return err
	}
	return RenderJSONFromBytes(w, r, data)
}

// renderJSONWithStatus sends data as json and enforces status code
func renderJSONWithStatus(w http.ResponseWriter, data interface{}, code int) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(true)
	if err := enc.Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_, _ = w.Write(buf.Bytes())
}

// ParseFromTo parses from and to query params of the request
func ParseFromTo(r *http.Request) (from, to time.Time, err error) {
	parseTimeStamp := func(ts string) (time.Time, error) {
		formats := []string{
			"2006-01-02T15:04:05.000000000",
			"2006-01-02T15:04:05",
			"2006-01-02T15:04",
			"20060102",
			time.RFC3339,
			time.RFC3339Nano,
		}

		for _, f := range formats {
			if t, e := time.Parse(f, ts); e == nil {
				return t, nil
			}
		}
		return time.Time{}, errors.Errorf("can't parse date %q", ts)
	}

	if from, err = parseTimeStamp(r.URL.Query().Get("from")); err != nil {
		return from, to, errors.Wrap(err, "incorrect from time")
	}

	if to, err = parseTimeStamp(r.URL.Query().Get("to")); err != nil {
		return from, to, errors.Wrap(err, "incorrect to time")
	}
	return from, to, nil
}
