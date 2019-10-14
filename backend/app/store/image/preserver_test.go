package image

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
)

func TestPreserver_ExtractExternalImages(t *testing.T) {
	table := []struct {
		desc    string
		input   string
		expcted []string
	}{
		{
			"Single external image",
			`<p> blah <img src="https://radio-t.com/img.png"/> test</p>`,
			[]string{"https://radio-t.com/img.png"},
		},
		{
			"Mupliple external images",
			`<p> blah <img src="https://radio-t.com/img1.png"/></br><img src="http://radio-t.com/img2.png"/> test</p>`,
			[]string{"https://radio-t.com/img1.png", "http://radio-t.com/img2.png"},
		},
		{
			"Single internal image",
			`<p> blah <img src="https://remark42.com/api/v1/picture/img.png"/> test</p>`,
			[]string{},
		},
		{
			"Not an img",
			`<p> blah <video src="https://radio-t.com/video.avi"/> test</p>`,
			[]string{},
		},
	}

	preserver := Preserver{Enabled: true, RemarkURL: "https://remark42.com"}
	for _, tt := range table {
		t.Run(tt.desc, func(t *testing.T) {
			res, err := preserver.extractExternalImages(tt.input)
			assert.Nil(t, err)
			assert.Equal(t, tt.expcted, res)
		})
	}
}

func TestPreserver_ConvertEnabled(t *testing.T) {
	httpServer := imgHTTPServer(t)
	defer httpServer.Close()

	table := []struct {
		desc     string
		mockInit func(mock *MockStore)
		input    string
		expected string
	}{
		{
			"Image successfuly replaced",
			func(mockStore *MockStore) {
				mockStore.On("Save", mock.Anything, mock.Anything, mock.Anything).Return("username/abc/xyz.png", nil)
			},
			fmt.Sprintf(`<img src="%s/image/img1.png"/>`, httpServer.URL),
			`<img src="https://remark42.com/api/v1/picture/username/abc/xyz.png"/>`,
		},
		{
			"All img entrances are replaced",
			func(mockStore *MockStore) {
				mockStore.On("Save", mock.Anything, mock.Anything, mock.Anything).Return("username/abc/xyz.png", nil)
			},
			fmt.Sprintf(`<img src="%s/image/img1.png"/><p>blah</p><img src="%s/image/img1.png"/>`, httpServer.URL, httpServer.URL),
			`<img src="https://remark42.com/api/v1/picture/username/abc/xyz.png"/><p>blah</p><img src="https://remark42.com/api/v1/picture/username/abc/xyz.png"/>`,
		},
		{
			"Internal image isn't replaced",
			func(mockStore *MockStore) {},
			`<img src="https://remark42.com/api/v1/picture/username/abc/xyz.png"/>`,
			`<img src="https://remark42.com/api/v1/picture/username/abc/xyz.png"/>`,
		},
		{
			"Not found image keeps original url",
			func(mockStore *MockStore) {},
			fmt.Sprintf(`<img src="%s/image/not_existing_img.png"/>`, httpServer.URL),
			fmt.Sprintf(`<img src="%s/image/not_existing_img.png"/>`, httpServer.URL),
		},
		{
			"Timed out image keeps original url",
			func(mockStore *MockStore) {},
			fmt.Sprintf(`<img src="%s/image/img_slow.png"/>`, httpServer.URL),
			fmt.Sprintf(`<img src="%s/image/img_slow.png"/>`, httpServer.URL),
		},
		{
			"Image failed to be saved keeps original url",
			func(mockStore *MockStore) {
				mockStore.On("Save", mock.Anything, mock.Anything, mock.Anything).Return("", errors.New(""))
			},
			fmt.Sprintf(`<img src="%s/image/img1.png"/>`, httpServer.URL),
			fmt.Sprintf(`<img src="%s/image/img1.png"/>`, httpServer.URL),
		},
	}

	for _, tt := range table {
		t.Run(tt.desc, func(t *testing.T) {
			mockStore := MockStore{}
			imgService := Service{Store: &mockStore}
			preserver := Preserver{Enabled: true, RemarkURL: "https://remark42.com", ImageService: &imgService, Timeout: 100 * time.Millisecond}
			tt.mockInit(&mockStore)
			assert.Equal(t, tt.expected, preserver.Convert(tt.input, "username"))
		})
	}
}

func TestPreserver_ConvertDisabled(t *testing.T) {
	preserver := Preserver{Enabled: false, RemarkURL: "https://remark42.com"}
	assert.Equal(t, `<img src="https://radio-t.com/img.png"/>`, preserver.Convert(`<img src="https://radio-t.com/img.png"/>`, "username"))
}

func imgHTTPServer(t *testing.T) *httptest.Server {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/image/img1.png" {
			t.Log("http img request", r.URL)
			w.Header().Add("Content-Length", "123")
			w.Header().Add("Content-Type", "image/png")
			_, err := w.Write([]byte(fmt.Sprintf("%123s", "X")))
			assert.NoError(t, err)
			return
		}
		if r.URL.Path == "/image/img_slow.png" {
			time.Sleep(200 * time.Millisecond)
			w.WriteHeader(500)
			return
		}
		t.Log("http img request - not found", r.URL)
		w.WriteHeader(404)
	}))

	return ts
}
