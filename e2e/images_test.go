//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// IMAGE_DISABLE is an instance-wide server option: it refuses POST /picture, strips images from
// new, previewed and edited comments, and reports image_disabled in /config so the widget drops
// its image controls. Every case below runs against two instances, the one started with the
// option and the main one, and the main one is what makes the first half mean anything: a
// selector that finds no image, a request that was never made or a 403 that came from somewhere
// else all look exactly like success on an instance where nothing happened.

// imageInstance is one of the two instances each case runs against
type imageInstance struct {
	name     string
	base     string
	disabled bool
}

var imageInstances = []imageInstance{
	{"images disabled", noImagesURL, true},
	{"default instance", baseURL, false},
}

// imageMarkdown is an external image, so nothing here depends on an upload having worked
const imageMarkdown = "![x](http://example.com/a.png)"

// signInImageReader signs in as a reader the widget offers an image uploader to. Anonymous
// readers never get one (root.tsx, comment.tsx), so on the disabled instance an anonymous reader
// would show no image controls with the option off, and the cases about those controls would
// prove nothing. tag keeps the address, and so the user, apart between cases: a user is the
// sha1 of the address and the mailbox outlives the run
func signInImageReader(t *testing.T, page playwright.Page, frame playwright.FrameLocator, in imageInstance, tag string) {
	t.Helper()
	if in.disabled {
		signInEmail(t, page, frame, "imagereader", fmt.Sprintf("images-%s-%s@example.com", tag, runID))
		return
	}
	signInDev(t, page, frame)
}

// wantImages is how many images the comment carrying imageMarkdown should show
func (in imageInstance) wantImages() int {
	if in.disabled {
		return 0
	}
	return 1
}

// imagesInComment counts the images in the text of the comment carrying marker. The avatar is an
// image too and sits outside the text, so counting every img in the article would never be zero
func imagesInComment(t *testing.T, frame playwright.FrameLocator, marker string) int {
	t.Helper()
	n, err := comment(frame, marker).Locator(".raw-content img").Count()
	require.NoError(t, err)
	return n
}

// TestImages_ExternalImageIsStrippedAndTheRestPosts covers the main promise: an image in a new
// comment does not come out, and the rest of the comment does. The picture is external, which is
// the case an upload limit could never cover. Checked on the posted comment and again after a
// reload, since the first is the response the widget just rendered and the second is what the
// server stored
func TestImages_ExternalImageIsStrippedAndTheRestPosts(t *testing.T) {
	t.Parallel()

	for _, in := range imageInstances {
		t.Run(in.name, func(t *testing.T) {
			t.Parallel()

			page := newPage(t)
			frame := openURL(t, page, threadURLOn(t, in.base))
			signInImageReader(t, page, frame, in, "external")

			marker := "kept text " + runID
			posted := postCommentMatching(t, frame, "before "+imageMarkdown+" "+marker, marker)

			text, err := posted.InnerText()
			require.NoError(t, err)
			assert.Contains(t, text, "before", "the text around the image was lost with it")
			assert.Equal(t, in.wantImages(), imagesInComment(t, frame, marker), "images in the posted comment")

			frame = reload(t, page)
			waitVisible(t, comment(frame, marker))
			assert.Equal(t, in.wantImages(), imagesInComment(t, frame, marker), "images in the stored comment")
		})
	}
}

// TestImages_PreviewShowsNoImage covers the preview, which is formatted by its own handler. A
// preview that kept the image would show the commenter something the posted comment will not have
func TestImages_PreviewShowsNoImage(t *testing.T) {
	t.Parallel()

	for _, in := range imageInstances {
		t.Run(in.name, func(t *testing.T) {
			t.Parallel()

			page := newPage(t)
			frame := openURL(t, page, threadURLOn(t, in.base))
			signInImageReader(t, page, frame, in, "preview")

			marker := "previewed text " + runID
			form := frame.Locator(commentFormSel).First()
			require.NoError(t, form.Locator("textarea").Fill("before "+imageMarkdown+" "+marker))
			require.NoError(t, form.Locator(`button:has-text("Preview")`).Click())

			preview := form.Locator(".raw-content", playwright.LocatorLocatorOptions{HasText: marker}).First()
			waitVisible(t, preview)
			n, err := preview.Locator("img").Count()
			require.NoError(t, err)
			assert.Equal(t, in.wantImages(), n, "images in the preview")
		})
	}
}

// TestImages_ImageOnlyCommentIsRejected covers a comment with nothing but an image, which has no
// text left to post. The server turns it down and the form says so. A first, ordinary comment is
// posted beforehand so that "nothing was added" is a count of one and not an empty thread, which
// is also what an unrendered one looks like
func TestImages_ImageOnlyCommentIsRejected(t *testing.T) {
	t.Parallel()

	for _, in := range imageInstances {
		t.Run(in.name, func(t *testing.T) {
			t.Parallel()

			page := newPage(t)
			frame := openURL(t, page, threadURLOn(t, in.base))
			signInImageReader(t, page, frame, in, "only")

			anchor := "anchor " + runID
			postComment(t, frame, anchor)

			form := frame.Locator(commentFormSel).First()
			submitForm(t, form, imageMarkdown)

			wantArticles := 2
			if in.disabled {
				wantArticles = 1
				waitVisible(t, form.Locator(`p[role="alert"]`))
			}

			// no wait on the anchor's text after the reload: on the default instance the image-only
			// comment sits above it and pushes it below the fold, where it is an empty placeholder
			// that no text filter matches. Counting articles sees placeholders too
			frame = reload(t, page)
			eventually(t, waitTimeout, fmt.Sprintf("the thread did not settle at %d comments", wantArticles), func() bool {
				n, err := frame.Locator("article").Count()
				return err == nil && n == wantArticles
			})

			// the newest comment is first and in view, so on the default instance its image is drawn
			// by now. Polled all the same, and for the disabled instance that is a poll for zero,
			// which holds from the start: the thread is one comment and it is the anchor
			eventually(t, waitTimeout, fmt.Sprintf("the thread did not settle at %d images", in.wantImages()), func() bool {
				n, err := frame.Locator("article .raw-content img").Count()
				return err == nil && n == in.wantImages()
			})
		})
	}
}

// TestImages_EditCannotAddAnImage covers the edit path, which has its own handler: an edit may
// not bring in an image the original did not have. Checked after a reload so it is the stored
// comment.
//
// The last part records what the edit form shows afterwards, and it is the same on both
// instances: the original markdown is stored as typed, so the form offers the image line back
// even though it is not rendered. Saving that again strips it. This documents the behavior the
// docs describe and is not a requirement, so a change that strips the markdown too should change
// this assertion, not work around it
func TestImages_EditCannotAddAnImage(t *testing.T) {
	t.Parallel()

	for _, in := range imageInstances {
		t.Run(in.name, func(t *testing.T) {
			t.Parallel()

			page := newPage(t)
			frame := openURL(t, page, threadURLOn(t, in.base))
			signInImageReader(t, page, frame, in, "edit")

			original := "edit me " + runID
			postComment(t, frame, original)

			require.NoError(t, actions(frame, original).Locator(`button:has-text("Edit")`).Click())
			edited := "edited " + runID
			submitForm(t, replyForm(t, frame), edited+" "+imageMarkdown)
			waitVisible(t, comment(frame, edited))

			frame = reload(t, page)
			waitVisible(t, comment(frame, edited))
			assert.Equal(t, in.wantImages(), imagesInComment(t, frame, edited), "images in the edited comment")

			require.NoError(t, actions(frame, edited).Locator(`button:has-text("Edit")`).Click())
			textarea := replyForm(t, frame).Locator("textarea")
			eventually(t, waitTimeout, "the edit form did not offer the markdown as it was typed", func() bool {
				v, err := textarea.InputValue()
				return err == nil && strings.Contains(v, imageMarkdown)
			})
		})
	}
}

// TestImages_UploadIsRefused covers the server refusing an upload on its own, whatever the widget
// shows: the controls are only politeness. A 403 alone would not say why, since a missing session
// or XSRF token is refused the same way, so the answer has to carry the option's own code and
// message. The default instance is the control: the same request there fails as an unparseable
// upload, which is not that.
//
// Two things have to be right for the request to reach the handler at all, and each otherwise
// ends in the same bare "Access denied" 403 on both instances: the reader is signed in with an
// account, since the route turns an anonymous user away first (rest.go, rejectAnonUser), and the
// url carries site=remark, which the middleware matches against the token's audience. The
// first run of this case missed the second and got that answer from both
func TestImages_UploadIsRefused(t *testing.T) {
	t.Parallel()

	for _, in := range imageInstances {
		t.Run(in.name, func(t *testing.T) {
			t.Parallel()

			page := newPage(t)
			frame := openURL(t, page, threadURLOn(t, in.base))
			signInImageReader(t, page, frame, in, "upload")

			status, body := pageFetch(t, page, "POST", in.base+"/api/v1/picture?site=remark", nil)
			if in.disabled {
				assert.Equal(t, http.StatusForbidden, status, "body: %s", body)
				assert.Contains(t, body, `"code":17`, "ErrActionRejected")
				assert.Contains(t, body, "image uploads are disabled")
				return
			}
			assert.NotEqual(t, http.StatusForbidden, status, "the default instance refused an upload: %s", body)
			assert.NotContains(t, body, "image uploads are disabled")
		})
	}
}

// TestImages_FormOffersNoImageControls covers the widget following the server. The toolbar is
// asserted present on both so that its image control missing is the image control and not the
// toolbar, and the setting is read back from /config so a field the widget does not read, or one
// the server renamed, is named here and not left to look like a rendering fault.
//
// Counted after a reload and only once the reader is shown as signed in: until then the form is
// drawn without a user and has no uploader either way, which would make the absence meaningless
func TestImages_FormOffersNoImageControls(t *testing.T) {
	t.Parallel()

	for _, in := range imageInstances {
		t.Run(in.name, func(t *testing.T) {
			t.Parallel()

			page := newPage(t)
			frame := openURL(t, page, threadURLOn(t, in.base))
			signInImageReader(t, page, frame, in, "controls")

			status, body := pageFetch(t, page, "GET", in.base+"/api/v1/config?site=remark", nil)
			require.Equal(t, http.StatusOK, status, body)
			var config struct {
				ImageDisabled bool `json:"image_disabled"`
			}
			require.NoError(t, json.Unmarshal([]byte(body), &config), body)
			assert.Equal(t, in.disabled, config.ImageDisabled, "image_disabled in /config")

			frame = reload(t, page)
			waitVisible(t, frame.Locator(`[title="Sign out"]`))
			form := frame.Locator(commentFormSel).First()
			// by element name, which the production bundle keeps, see TestWidgets_SimpleViewHidesTheEditingFurniture
			waitVisible(t, form.Locator("md-bold"))

			upload := form.Locator(`input[type="file"]`)
			if !in.disabled {
				eventually(t, waitTimeout, "the default instance offered no image control", func() bool {
					n, err := upload.Count()
					return err == nil && n == 1
				})
				return
			}
			n, err := upload.Count()
			require.NoError(t, err)
			assert.Zero(t, n, "the widget kept its image control on an instance with images disabled")
		})
	}
}

// imageEvents is run on the textarea. It drops an image on it, or pastes one, the way a reader
// would, and reports whether each event was canceled, which is the handler having claimed it.
// It waits two frames before returning so that anything the handlers scheduled has happened
const imageEvents = `async (el, kind) => {
	const dt = new DataTransfer();
	dt.items.add(new File([new Uint8Array([137, 80, 78, 71])], 'a.png', {type: 'image/png'}));
	const init = {dataTransfer: dt, bubbles: true, cancelable: true};
	const out = {};
	if (kind === 'drop') {
		out.dragoverCancelled = !el.dispatchEvent(new DragEvent('dragover', init));
		out.canceled = !el.dispatchEvent(new DragEvent('drop', init));
	} else {
		out.canceled = !el.dispatchEvent(new ClipboardEvent('paste', {clipboardData: dt, bubbles: true, cancelable: true}));
	}
	await new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve)));
	return out;
}`

// dispatchImageEvents runs imageEvents on the form's textarea and returns what it reported
func dispatchImageEvents(t *testing.T, frame playwright.FrameLocator, kind string) map[string]any {
	t.Helper()
	res, err := frame.Locator(commentFormSel).First().Locator("textarea").Evaluate(imageEvents, kind)
	require.NoError(t, err)
	out, ok := res.(map[string]any)
	require.True(t, ok, "unexpected shape from the page: %#v", res)
	return out
}

// TestImages_DropAndPasteAreIgnored covers a reader dragging or pasting an image into the form.
// With the option on, nothing is uploaded and the paste is left to the browser, while the drop is
// canceled so the frame is not sent to the file. The events are dispatched from script, since a
// file cannot be dragged in from outside the browser, so what this can see is the handler's
// answer and the requests it makes, and not the frame navigating, which is what canceling the
// drop is for. The default instance is the control: the same events there reach the handler and
// start uploads, which is what shows they arrive at all
func TestImages_DropAndPasteAreIgnored(t *testing.T) {
	t.Parallel()

	for _, in := range imageInstances {
		t.Run(in.name, func(t *testing.T) {
			t.Parallel()

			page := newPage(t)
			frame := openURL(t, page, threadURLOn(t, in.base))
			signInImageReader(t, page, frame, in, "drop")

			var uploads atomic.Int32
			page.OnRequest(func(r playwright.Request) {
				if strings.Contains(r.URL(), "/api/v1/picture") {
					uploads.Add(1)
				}
			})

			drop := dispatchImageEvents(t, frame, "drop")
			assert.Equal(t, true, drop["dragoverCancelled"], "the drag over the form was not claimed")
			assert.Equal(t, true, drop["canceled"], "the drop was left to the browser, which would open the file")

			paste := dispatchImageEvents(t, frame, "paste")

			if in.disabled {
				assert.Equal(t, false, paste["canceled"], "a pasted image was claimed by a form that takes none")
				assert.Zero(t, uploads.Load(), "an image was uploaded to an instance with images disabled")
				return
			}
			assert.Equal(t, true, paste["canceled"], "the default instance did not claim a pasted image")
			eventually(t, waitTimeout, "the default instance never started an upload", func() bool {
				return uploads.Load() >= 2
			})
		})
	}
}

// TestImages_AnonymousDropShowsNoSignInPrompt covers the order of the checks in the drop handler.
// Anonymous readers are told to sign in to attach images, which is wrong where no account can: the
// disabled check has to come first. The default instance is the control and has to show the
// prompt, otherwise its absence on the other says nothing about the order
func TestImages_AnonymousDropShowsNoSignInPrompt(t *testing.T) {
	t.Parallel()

	for _, in := range imageInstances {
		t.Run(in.name, func(t *testing.T) {
			t.Parallel()

			page := newPage(t)
			frame := openURL(t, page, threadURLOn(t, in.base))
			signInAnon(t, page, frame, anonName("imagedrop"))

			prompt := frame.Locator("text=Anonymous users cannot upload images")
			dispatchImageEvents(t, frame, "drop")

			if !in.disabled {
				waitVisible(t, prompt)
				return
			}
			n, err := prompt.Count()
			require.NoError(t, err)
			assert.Zero(t, n, "an anonymous reader was told to sign in for something no account can do")
		})
	}
}
