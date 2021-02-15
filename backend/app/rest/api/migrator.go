package api

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/go-chi/render"
	cache "github.com/go-pkgz/lcw"
	log "github.com/go-pkgz/lgr"
	R "github.com/go-pkgz/rest"
	"github.com/pkg/errors"

	"github.com/umputun/remark42/backend/app/migrator"
	"github.com/umputun/remark42/backend/app/rest"
)

// Migrator rest with import and export controllers
type Migrator struct {
	Cache             LoadingCache
	NativeImporter    migrator.Importer
	DisqusImporter    migrator.Importer
	WordPressImporter migrator.Importer
	NativeExporter    migrator.Exporter
	URLMapperMaker    migrator.MapperMaker
	KeyStore          KeyStore

	busy map[string]bool
	lock sync.Mutex
}

// KeyStore defines sub-interface for consumers needed just a key
type KeyStore interface {
	Key(siteID string) (key string, err error)
}

// POST /import?secret=key&site=site-id&provider=disqus|remark|wordpress
// imports comments from post body.
func (m *Migrator) importCtrl(w http.ResponseWriter, r *http.Request) {

	siteID := r.URL.Query().Get("site")

	if m.isBusy(siteID) {
		rest.SendErrorJSON(w, r, http.StatusConflict, errors.New("already running"),
			"import rejected", rest.ErrActionRejected)
		return
	}

	tmpfile, err := m.saveTemp(r.Body)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't save request to temp file", rest.ErrInternal)
		return
	}

	go m.runImport(siteID, r.URL.Query().Get("provider"), tmpfile) // import runs in background and sets busy flag for site

	render.Status(r, http.StatusAccepted)
	render.JSON(w, r, R.JSON{"status": "import request accepted"})
}

// POST /import/form?secret=key&site=site-id&provider=disqus|remark|wordpress
// imports comments from form body.
func (m *Migrator) importFormCtrl(w http.ResponseWriter, r *http.Request) {
	siteID := r.URL.Query().Get("site")

	if m.isBusy(siteID) {
		rest.SendErrorJSON(w, r, http.StatusConflict, errors.New("already running"),
			"import rejected", rest.ErrActionRejected)
		return
	}

	if err := r.ParseMultipartForm(20 * 1024 * 1024); err != nil { // 20M max memory, if bigger will make a file
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't parse multipart form", rest.ErrDecode)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't get import file from the request", rest.ErrInternal)
		return
	}
	defer func() { _ = file.Close() }()

	tmpfile, err := m.saveTemp(file)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't save request to temp file", rest.ErrInternal)
		return
	}

	go m.runImport(siteID, r.URL.Query().Get("provider"), tmpfile) // import runs in background and sets busy flag for site

	render.Status(r, http.StatusAccepted)
	render.JSON(w, r, R.JSON{"status": "import request accepted"})
}

// GET /wait?site=site-id
// waits for migration operation (import or remap)
func (m *Migrator) waitCtrl(w http.ResponseWriter, r *http.Request) {
	siteID := r.URL.Query().Get("site")
	timeOut := time.Minute * 15
	if v := r.URL.Query().Get("timeout"); v != "" {
		if vv, e := time.ParseDuration(v); e == nil {
			timeOut = vv
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeOut)
	defer cancel()
	for {
		if !m.isBusy(siteID) {
			break
		}
		select {
		case <-ctx.Done():
			render.Status(r, http.StatusGatewayTimeout)
			render.JSON(w, r, R.JSON{"status": "timeout expired", "site_id": siteID})
			return
		case <-time.After(100 * time.Millisecond):
		}
	}
	render.Status(r, http.StatusOK)
	render.JSON(w, r, R.JSON{"status": "completed", "site_id": siteID})
}

// GET /export?site=site-id&secret=12345&?mode=file|stream
// exports all comments for siteID as gz file
func (m *Migrator) exportCtrl(w http.ResponseWriter, r *http.Request) {

	siteID := r.URL.Query().Get("site")

	var writer io.Writer = w
	if r.URL.Query().Get("mode") == "file" {
		exportFile := fmt.Sprintf("%s-%s.json.gz", siteID, time.Now().Format("20060102"))
		w.Header().Set("Content-Type", "application/gzip")
		w.Header().Set("Content-Disposition", "attachment;filename="+exportFile)
		w.WriteHeader(http.StatusOK)
		gzWriter := gzip.NewWriter(w)
		defer func() {
			if e := gzWriter.Close(); e != nil {
				log.Printf("[WARN] can't close gzip writer, %s", e)
			}
		}()
		writer = gzWriter
	}

	if _, err := m.NativeExporter.Export(writer, siteID); err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "export failed", rest.ErrInternal)
		return
	}
}

// POST /remap?site=site-id
// remap urls in comments based on given rules (oldUrl newUrl)
func (m *Migrator) remapCtrl(w http.ResponseWriter, r *http.Request) {
	siteID := r.URL.Query().Get("site")

	// create new url-mapper from given rules in body
	mapper, err := m.URLMapperMaker(r.Body)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "remap failed, bad given rules", rest.ErrDecode)
		return
	}
	defer r.Body.Close()

	// start remap procedure with mapper
	go func() {
		m.setBusy(siteID, true)
		defer m.setBusy(siteID, false)

		// do export
		fh, e := ioutil.TempFile("", "remark42_convert")
		if e != nil {
			log.Printf("[WARN] failed to make temp file %+v", e)
			return
		}
		defer func() {
			if e = os.Remove(fh.Name()); e != nil {
				log.Printf("[WARN] failed to remove temp file %+v", e)
			}
		}()
		log.Printf("[DEBUG] start export for site=%s", siteID)
		if _, e = m.NativeExporter.Export(fh, siteID); e != nil {
			log.Printf("[WARN] export failed with %+v", e)
			return
		}

		if _, e = fh.Seek(0, 0); e != nil {
			log.Printf("[WARN] failed to seek file %+v", e)
			return
		}

		log.Printf("[DEBUG] start import for site=%s", siteID)
		mappedReader := migrator.WithMapper(fh, mapper)
		size, e := m.NativeImporter.Import(mappedReader, siteID)
		if e != nil {
			log.Printf("[WARN] import failed with %+v", e)
			return
		}

		m.Cache.Flush(cache.Flusher(siteID).Scopes(siteID))
		log.Printf("[DEBUG] convert request completed. site=%s, comments=%d", siteID, size)
	}()

	render.Status(r, http.StatusAccepted)
	render.JSON(w, r, R.JSON{"status": "convert request accepted"})
}

// runImport reads from tmpfile and import for given siteID and provider
func (m *Migrator) runImport(siteID, provider, tmpfile string) {
	m.setBusy(siteID, true)

	defer func() {
		m.setBusy(siteID, false)
		if err := os.Remove(tmpfile); err != nil {
			log.Printf("[WARN] failed to remove tmp file %s, %v", tmpfile, err)
		}
	}()

	var importer migrator.Importer
	switch provider {
	case "disqus":
		importer = m.DisqusImporter
	case "wordpress":
		importer = m.WordPressImporter
	default:
		importer = m.NativeImporter
	}
	log.Printf("[DEBUG] import request for site=%s, provider=%s", siteID, provider)

	fh, err := os.Open(tmpfile) // nolint
	if err != nil {
		log.Printf("[WARN] import failed, %v", err)
		return
	}

	size, err := importer.Import(fh, siteID)
	if err != nil {
		log.Printf("[WARN] import failed, %v", err)
		return
	}
	m.Cache.Flush(cache.Flusher(siteID).Scopes(siteID))
	log.Printf("[DEBUG] import request completed. site=%s, provider=%s, comments=%d", siteID, provider, size)
}

// saveTemp reads from reader and saves to temp file
func (m *Migrator) saveTemp(r io.Reader) (string, error) {
	tmpfile, err := ioutil.TempFile("", "remark42_import")
	if err != nil {
		return "", errors.Wrap(err, "can't make temp file")
	}

	if _, err = io.Copy(tmpfile, r); err != nil {
		return "", errors.Wrap(err, "can't copy to temp file")
	}

	if err = tmpfile.Close(); err != nil {
		return "", errors.Wrap(err, "can't close temp file")
	}

	return tmpfile.Name(), nil
}

// isBusy checks busy flag from the map by siteID as key
func (m *Migrator) isBusy(siteID string) bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.busy == nil {
		m.busy = map[string]bool{}
	}
	return m.busy[siteID]
}

// setBusy sets/resets busy flag to the map by siteID as key
func (m *Migrator) setBusy(siteID string, status bool) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.busy == nil {
		m.busy = map[string]bool{}
	}
	m.busy[siteID] = status
}
