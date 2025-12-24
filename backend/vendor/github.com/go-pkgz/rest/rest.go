// Package rest provides common middlewares and helpers for rest services
package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// JSON is a map alias, just for convenience
type JSON map[string]any

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
		return fmt.Errorf("failed to send response to %s: %w", r.RemoteAddr, err)
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
			return nil, fmt.Errorf("json encoding failed: %w", err)
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
		return time.Time{}, fmt.Errorf("can't parse date %q", ts)
	}

	if from, err = parseTimeStamp(r.URL.Query().Get("from")); err != nil {
		return from, to, fmt.Errorf("incorrect from time: %w", err)
	}

	if to, err = parseTimeStamp(r.URL.Query().Get("to")); err != nil {
		return from, to, fmt.Errorf("incorrect to time: %w", err)
	}
	return from, to, nil
}

// DecodeJSON decodes json request from http.Request to given type
func DecodeJSON[T any](r *http.Request, res *T) error {
	if err := json.NewDecoder(r.Body).Decode(&res); err != nil {
		return fmt.Errorf("decode json: %w", err)
	}
	return nil
}

// EncodeJSON encodes given type to http.ResponseWriter and sets status code and content type header
func EncodeJSON[T any](w http.ResponseWriter, status int, v T) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}
