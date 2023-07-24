package search

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/blevesearch/bleve/v2"

	analyzerCustom "github.com/blevesearch/bleve/v2/analysis/analyzer/custom"
	analyzerStandard "github.com/blevesearch/bleve/v2/analysis/analyzer/standard"

	// not all currently supported locales are supported by bleve, import only those that are
	langAr "github.com/blevesearch/bleve/v2/analysis/lang/ar"
	langDe "github.com/blevesearch/bleve/v2/analysis/lang/de"
	langEn "github.com/blevesearch/bleve/v2/analysis/lang/en"
	langEs "github.com/blevesearch/bleve/v2/analysis/lang/es"
	langFi "github.com/blevesearch/bleve/v2/analysis/lang/fi"
	langFr "github.com/blevesearch/bleve/v2/analysis/lang/fr"
	langIt "github.com/blevesearch/bleve/v2/analysis/lang/it"
	langRu "github.com/blevesearch/bleve/v2/analysis/lang/ru"

	"github.com/blevesearch/bleve/v2/analysis/token/lowercase"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/umputun/remark42/backend/app/store"
)

var analyzerNameMapping = map[string]string{
	"standard": analyzerStandard.Name,
	"ar":       langAr.AnalyzerName,
	"de":       langDe.AnalyzerName,
	"en":       langEn.AnalyzerName,
	"es":       langEs.AnalyzerName,
	"fi":       langFi.AnalyzerName,
	"fr":       langFr.AnalyzerName,
	"it":       langIt.AnalyzerName,
	"ru":       langRu.AnalyzerName,
}

type bleveEngine struct {
	index     bleve.Index
	indexPath string
}

// Index adds set of comments to the index
func (b *bleveEngine) Index(comments []store.Comment) error {
	batch := b.index.NewBatch()
	for _, comment := range comments {
		key := DocumentKey{Locator: comment.Locator, ID: comment.ID}
		keyBytes, err := json.Marshal(key)
		if err != nil {
			return fmt.Errorf("can't marshal key %+v: %w", key, err)
		}

		doc := newBleveDoc(comment)
		err = batch.Index(string(keyBytes), doc)
		if err != nil {
			return fmt.Errorf("can't add to indexing batch: %w", err)
		}
	}
	err := b.index.Batch(batch)
	if err != nil {
		return fmt.Errorf("index error: %w", err)
	}

	return nil
}

// Search performs search request
func (b *bleveEngine) Search(req *Request) (*Result, error) {
	bQuery := bleve.NewQueryStringQuery(req.Query)
	bReq := bleve.NewSearchRequestOptions(bQuery, req.Limit, req.Skip, false)

	switch {
	case req.SortBy == "":
		bReq.SortBy([]string{"-time"})
	case req.SortBy == "time" || req.SortBy == "-time" || req.SortBy == "+time":
		bReq.SortBy([]string{req.SortBy})
	default:
		return nil, fmt.Errorf("unknown sort field %s", req.SortBy)
	}

	searchRes, err := b.index.Search(bReq)
	if err != nil {
		return nil, fmt.Errorf("search error: %w", err)
	}

	log.Printf("[INFO] found %d documents for query %q in %s",
		searchRes.Total, req.Query, searchRes.Took.String())

	result := convertResultRepr(searchRes)
	return &result, nil
}

// Size is the number of documents in the index
func (b *bleveEngine) Size() (uint64, error) {
	return b.index.DocCount()
}

// Close closes index
func (b *bleveEngine) Close() error {
	return b.index.Close()
}

func newBleveEngine(indexPath, analyzer string) (*bleveEngine, error) {
	indexExists, err := checkIndexPath(indexPath)
	if err != nil {
		return nil, err
	}

	var index bleve.Index
	if !indexExists {
		log.Printf("[INFO] creating new search index %s", indexPath)

		bleveAnalyzerName, has := analyzerNameMapping[analyzer]
		if !has {
			return nil, fmt.Errorf("unknown analyzer %q", analyzer)
		}
		index, err = bleve.New(indexPath, createIndexMapping(bleveAnalyzerName))
		if err != nil {
			return nil, fmt.Errorf("cannot open index: %w", err)
		}
	} else {
		log.Printf("[INFO] opening existing search index %s", indexPath)
		index, err = bleve.Open(indexPath)
		if err != nil {
			return nil, fmt.Errorf("cannot open index: %w", err)
		}
	}

	return &bleveEngine{
		index:     index,
		indexPath: indexPath,
	}, nil
}

// checkIndexPath creates directories if path is not exists and checks if it is empty
// returns true if there already was some files in index path
func checkIndexPath(path string) (bool, error) {
	st, err := os.Stat(path)

	if err != nil && !os.IsNotExist(err) {
		return false, err
	}

	if os.IsNotExist(err) {
		err = os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return false, err
		}
		return false, nil
	}

	if !st.IsDir() {
		return false, fmt.Errorf("index path should be a directory")
	}
	// check if `path` directory is empty
	f, err := os.Open(path) //nolint:gosec // opening directory in `SEARCH_INDEX_PATH`
	if err != nil {
		return false, err
	}
	defer func() { _ = f.Close() }()

	names, err := f.Readdirnames(1)
	if err != nil && err != io.EOF {
		return false, err
	}
	// some files are found, consider index as not empty
	return len(names) > 0, nil
}

// bleveDoc is the structure that would be indexed.
// We index only these fields from the comment.
// Actually, we can index store.Comment directly.
// but we want to exclude unnecessary fields and
// change the layout of the document, e.g.
// have 'user' field instead of 'user.name'.
// The bleve doen't support mapping of the nested fields to top-level
// see https://github.com/blevesearch/bleve/issues/229
type bleveDoc struct {
	Text      string    `json:"text"`
	User      string    `json:"user"`
	Timestamp time.Time `json:"time"`
}

func newBleveDoc(comment store.Comment) *bleveDoc {
	return &bleveDoc{
		Text:      comment.Text,
		User:      comment.User.Name,
		Timestamp: comment.Timestamp,
	}
}

func textMapping(analyzer string, doStore bool) *mapping.FieldMapping {
	textFieldMapping := bleve.NewTextFieldMapping()
	textFieldMapping.Store = doStore
	textFieldMapping.Analyzer = analyzer
	return textFieldMapping
}

func createIndexMapping(textAnalyzer string) mapping.IndexMapping {
	indexMapping := bleve.NewIndexMapping()

	err := indexMapping.AddCustomAnalyzer("keyword_lower", map[string]interface{}{
		"type": analyzerCustom.Name, "tokenizer": "single", "token_filters": []string{lowercase.Name},
	})
	if err != nil {
		panic(fmt.Sprintf("error adding bleve analyzer %v", err))
	}

	// setup how comments would be indexed
	commentMapping := bleve.NewDocumentMapping()
	commentMapping.AddFieldMappingsAt("text", textMapping(textAnalyzer, false))
	commentMapping.AddFieldMappingsAt("user", textMapping("keyword_lower", true))
	commentMapping.AddFieldMappingsAt("time", bleve.NewDateTimeFieldMapping())

	indexMapping.AddDocumentMapping("_default", commentMapping)

	return indexMapping
}

// convertBleveSerp converts bleve search result to search.Result
func convertResultRepr(bleveResult *bleve.SearchResult) Result {
	result := Result{
		Total: bleveResult.Total,
		Keys:  make([]DocumentKey, 0, len(bleveResult.Hits)),
	}

	for _, r := range bleveResult.Hits {
		key := DocumentKey{}
		err := json.Unmarshal([]byte(r.ID), &key)
		if err != nil {
			log.Printf("[WARN] can't unmarshal key %s, %s", r.ID, err)
			continue
		}
		result.Keys = append(result.Keys, key)
	}
	return result
}
