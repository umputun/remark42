package rewrite

import "net/http"
import "net/url"
import "strings"
import "regexp"
import "path"
import "fmt"

const headerField = "X-Rewrite-Original-URI"

type Rule struct {
	Pattern string
	To      string
	*regexp.Regexp
}

var regfmt = regexp.MustCompile(`:[^/#?()\.\\]+`)

func NewRule(pattern, to string) (*Rule, error) {
	pattern = regfmt.ReplaceAllStringFunc(pattern, func(m string) string {
		return fmt.Sprintf(`(?P<%s>[^/#?]+)`, m[1:])
	})

	reg, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	return &Rule{
		pattern,
		to,
		reg,
	}, nil
}

func (r *Rule) Rewrite(req *http.Request) bool {
	oriPath := req.URL.Path

	if !r.MatchString(oriPath) {
		return false
	}

	to := path.Clean(r.Replace(req.URL))

	u, e := url.Parse(to)
	if e != nil {
		return false
	}

	req.Header.Set(headerField, req.URL.RequestURI())

	req.URL.Path = u.Path
	req.URL.RawPath = u.RawPath
	if u.RawQuery != "" {
		req.URL.RawQuery = u.RawQuery
	}

	return true
}

func (r *Rule) Replace(u *url.URL) string {
	if !hit("\\$|\\:", r.To) {
		return r.To
	}

	uri := u.RequestURI()

	regFrom := regexp.MustCompile(r.Pattern)
	match := regFrom.FindStringSubmatchIndex(uri)

	result := regFrom.ExpandString([]byte(""), r.To, uri, match)

	str := string(result[:])

	if hit("\\:", str) {
		return r.replaceNamedParams(uri, str)
	}

	return str
}

var urlreg = regexp.MustCompile(`:[^/#?()\.\\]+|\(\?P<[a-zA-Z0-9]+>.*\)`)

func (r *Rule) replaceNamedParams(from, to string) string {
	fromMatches := r.FindStringSubmatch(from)

	if len(fromMatches) > 0 {
		for i, name := range r.SubexpNames() {
			if len(name) > 0 {
				to = strings.Replace(to, ":"+name, fromMatches[i], -1)
			}
		}
	}

	return to
}

func NewHandler(rules map[string]string) RewriteHandler {
	var h RewriteHandler

	for key, val := range rules {
		r, e := NewRule(key, val)
		if e != nil {
			panic(e)
		}

		h.rules = append(h.rules, r)
	}

	return h
}

type RewriteHandler struct {
	rules []*Rule
}

func (h *RewriteHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	for _, r := range h.rules {
		ok := r.Rewrite(req)
		if ok {
			break
		}
	}
}

func hit(pattern, str string) bool {
	r, e := regexp.MatchString(pattern, str)
	if e != nil {
		return false
	}

	return r
}
