//go:build e2e

package e2e

import (
	"encoding/json"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fieldState is what the comment field holds: its text, its selection and whether it has the focus
type fieldState struct {
	Value   string `json:"value"`
	Start   int    `json:"start"`
	End     int    `json:"end"`
	Focused bool   `json:"focused"`
}

func readField(t *testing.T, field playwright.Locator) fieldState {
	t.Helper()
	raw, err := field.Evaluate(`el => JSON.stringify({
		value: el.value, start: el.selectionStart, end: el.selectionEnd, focused: el.ownerDocument.activeElement === el,
	})`, nil)
	require.NoError(t, err)
	var state fieldState
	require.NoError(t, json.Unmarshal([]byte(raw.(string)), &state))
	return state
}

// waitField waits for the field to hold the text with this selection, and the focus
func waitField(t *testing.T, field playwright.Locator, want fieldState, msg string) {
	t.Helper()
	deadline := time.Now().Add(waitTimeout)
	for {
		got := readField(t, field)
		if got == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s: the field holds %+v, want %+v", msg, got, want)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// groupsTyping says whether the engine holds typing and an insert made after it as one undo step,
// which WebKit does on macOS and nowhere else the suite runs
func groupsTyping(engine string) bool {
	return engine == "webkit" && runtime.GOOS == "darwin"
}

// typedAlongTheWay lists every text a field held while text was typed into it one key at a time,
// from the empty field on, and then the text it was edited into
func typedAlongTheWay(text, edited string) []string {
	keys := []rune(text)
	typed := make([]string, 0, len(keys)+2)
	for i := range keys {
		typed = append(typed, string(keys[:i]))
	}
	return append(typed, text, edited)
}

// waitUndoneToTyped waits for undo to leave the field holding one of the texts typed along the way,
// with the focus, and returns that text. WebKit on macOS ends an undo step wherever its timing put
// the end, so which one is not fixed there
func waitUndoneToTyped(t *testing.T, field playwright.Locator, typed []string, msg string) string {
	t.Helper()
	deadline := time.Now().Add(waitTimeout)
	for {
		got := readField(t, field)
		if got.Focused && slices.Contains(typed, got.Value) {
			return got.Value
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s: the field holds %+v, want the focus and one of %q", msg, got, typed)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// waitValue waits for the field to hold the text
func waitValue(t *testing.T, field playwright.Locator, want, msg string) {
	t.Helper()
	eventually(t, waitTimeout, msg, func() bool { return readField(t, field).Value == want })
}

// TestCommentForm_CaretAndUndoSurviveTheToolbar types into the middle of the comment field and
// uses a toolbar button, in every engine. The form draws itself again on every key, and drawing
// must leave the caret where the reader put it; the toolbar's edit must stay in the browser's own
// undo history, and the text undo brings back must be the text the form sends.
//
// WebKit on macOS takes some or all of the typing before the edit back in the same step as the
// edit, as it did with the toolbar package, and where that step starts depends on timing: run
// alone, the first undo empties the field; in a loaded parallel run it took back the last key and
// the edit. There undo only has to drop the bold and leave a text typed along the way, and the
// comment is sent from every other engine and OS. Measured on a bare page with a textarea, the
// package's insert and no widget: Chromium, Firefox and WebKit on Linux (the playwright image of
// the version this module drives) take back the insert alone.
func TestCommentForm_CaretAndUndoSurviveTheToolbar(t *testing.T) {
	t.Parallel()

	forEachEngine(t, func(t *testing.T, page playwright.Page) {
		engine := page.Context().Browser().BrowserType().Name()
		frame := openURL(t, page, renderURL(t))
		signInAnon(t, page, frame, anonName("toolbar"+engine))
		form := frame.Locator(commentFormSel).First()
		field := form.Locator("textarea")

		// a marker to find the comment by, typed first, so nothing is typed after the undo
		marker := "undone " + engine + " " + runID + " "
		at := len(marker)
		require.NoError(t, field.Click())
		require.NoError(t, field.PressSequentially(marker+"helloworld"))
		for range 5 {
			require.NoError(t, field.Press("ArrowLeft"))
		}
		require.NoError(t, field.PressSequentially(" "))
		waitField(t, field, fieldState{Value: marker + "hello world", Start: at + 6, End: at + 6, Focused: true},
			"typing in the middle moved the caret")

		_, err := field.Evaluate(`(el, at) => el.setSelectionRange(at + 6, at + 11)`, at)
		require.NoError(t, err)
		require.NoError(t, form.Locator("md-bold").Click())
		waitField(t, field, fieldState{Value: marker + "hello **world**", Start: at + 8, End: at + 13, Focused: true},
			"the bold button did not wrap the selection")

		undone := fieldState{Value: marker + "hello world", Start: at + 6, End: at + 11, Focused: true}
		require.NoError(t, field.Press("ControlOrMeta+z"))
		if groupsTyping(engine) {
			undone.Value = waitUndoneToTyped(t, field, typedAlongTheWay(marker+"helloworld", marker+"hello world"),
				"undo did not take the bold back to a text typed before it")
		} else {
			waitField(t, field, undone, "undo did not take the bold back")
		}
		require.NoError(t, field.Press("ControlOrMeta+Shift+z"))
		waitValue(t, field, marker+"hello **world**", "redo did not put the bold back")
		require.NoError(t, field.Press("ControlOrMeta+z"))
		waitValue(t, field, undone.Value, "undo did not take the bold back a second time")
		if groupsTyping(engine) {
			return
		}

		// sent straight after the undo: the form has to hold what undo left, without the bold
		require.NoError(t, form.Locator(`button[type="submit"]`).Click())
		posted := comment(frame, strings.TrimSpace(marker))
		waitVisible(t, posted)
		text, err := posted.Locator(".raw-content").InnerText()
		require.NoError(t, err)
		assert.Equal(t, marker+"hello world", text)
		n, err := posted.Locator(".raw-content strong").Count()
		require.NoError(t, err)
		assert.Zero(t, n, "the comment carries the bold that undo took away")
	})
}

// TestCommentForm_EmojiCompletionInsertsWhatIsSent turns the emoji completion on for one page and
// completes a name from the list, in every engine. The list has to open under the field, move with
// the arrows and put the chosen name in on Enter, and the comment sent right after a completion has
// to carry it. Undo and redo then have to take a completion back and put it back; WebKit on macOS
// takes some or all of the typing before the completion back with it, as it does for the toolbar,
// so there undo only has to leave a text typed along the way.
func TestCommentForm_EmojiCompletionInsertsWhatIsSent(t *testing.T) {
	t.Parallel()

	forEachEngine(t, func(t *testing.T, page playwright.Page) {
		engine := page.Context().Browser().BrowserType().Name()
		// the stack runs without the emoji option; the server's answer is changed for this page alone
		require.NoError(t, page.Route("**/api/v1/config?**", func(route playwright.Route) {
			resp, err := route.Fetch()
			if err != nil {
				_ = route.Abort()
				return
			}
			var config map[string]any
			body, err := resp.Body()
			if err != nil || json.Unmarshal(body, &config) != nil {
				_ = route.Fulfill(playwright.RouteFulfillOptions{Response: resp})
				return
			}
			config["emoji_enabled"] = true
			changed, _ := json.Marshal(config)
			_ = route.Fulfill(playwright.RouteFulfillOptions{Response: resp, Body: string(changed)})
		}))
		frame := openURL(t, page, renderURL(t))
		signInAnon(t, page, frame, anonName("emoji"+engine))
		form := frame.Locator(commentFormSel).First()
		field := form.Locator("textarea")

		// a marker to find the comment by, typed first, so nothing is typed after the completion;
		// the name goes on the second line, where the list has to follow the caret
		marker := "emoji " + engine + " " + runID
		prefix := marker + "\nhi :"
		require.NoError(t, field.Click())
		require.NoError(t, field.PressSequentially(marker))
		require.NoError(t, field.Press("Enter"))
		require.NoError(t, field.PressSequentially("hi :smi"))
		options := form.Locator(`[role="option"]`)
		waitVisible(t, options.First())
		names, err := options.EvaluateAll(`items => items.map((item) => item.dataset.value)`)
		require.NoError(t, err)
		list, ok := names.([]any)
		require.True(t, ok)
		require.GreaterOrEqual(t, len(list), 2, "fewer than two names start with smi")
		selected, err := options.First().GetAttribute("aria-selected")
		require.NoError(t, err)
		assert.Equal(t, "true", selected, "the first name is not selected")

		// the list opens under the second line, at the colon, where it would not be if nothing
		// placed it: its stylesheet puts it at the field's corner
		fieldBox, err := field.BoundingBox()
		require.NoError(t, err)
		listBox, err := form.Locator(`ul:has([role="option"])`).BoundingBox()
		require.NoError(t, err)
		// where the caret after the colon is drawn: past the field's padding, the width of "hi :" in
		// its font, and the bottom of the second line
		raw, err := field.Evaluate(`el => {
			const style = getComputedStyle(el);
			const context = document.createElement('canvas').getContext('2d');
			context.font = style.font;
			return JSON.stringify({
				line: parseFloat(style.lineHeight),
				top: parseFloat(style.paddingTop) + parseFloat(style.borderTopWidth),
				left: parseFloat(style.paddingLeft) + parseFloat(style.borderLeftWidth),
				text: context.measureText('hi :').width,
			});
		}`, nil)
		require.NoError(t, err)
		var metrics struct{ Line, Top, Left, Text float64 }
		require.NoError(t, json.Unmarshal([]byte(raw.(string)), &metrics))
		wantX := fieldBox.X + metrics.Left + metrics.Text
		wantY := fieldBox.Y + metrics.Top + 2*metrics.Line
		assert.InDelta(t, wantX, listBox.X, 6, "the list is not at the colon")
		assert.InDelta(t, wantY, listBox.Y, metrics.Line/2, "the list is not under the second line")

		require.NoError(t, field.Press("ArrowDown"))
		require.NoError(t, field.Press("Enter"))
		second := list[1].(string)
		completed := prefix + second + ": "
		waitField(t, field, fieldState{Value: completed, Start: len(completed), End: len(completed), Focused: true},
			"Enter did not put the second name in")
		waitHidden(t, options.First(), "the list stayed open after the completion")

		// sent at once, before anything else touches the field: the form has to hold what the
		// completion put in
		require.NoError(t, form.Locator(`button[type="submit"]`).Click())
		posted := comment(frame, marker)
		waitVisible(t, posted)
		text, err := posted.Locator(".raw-content").InnerText()
		require.NoError(t, err)
		assert.Equal(t, marker+" hi :"+second+":", strings.Join(strings.Fields(text), " "))

		// a second completion, which undo takes back and redo puts back. It is made on a new page:
		// once the form has emptied the field from script, the undo history of WebKit on macOS no
		// longer matches the text, with the package as without it
		waitValue(t, field, "", "the field was not emptied once the comment was sent")
		frame = reload(t, page)
		form = frame.Locator(commentFormSel).First()
		field = form.Locator("textarea")
		options = form.Locator(`[role="option"]`)
		require.NoError(t, field.Click())
		require.NoError(t, field.PressSequentially("again :smi"))
		waitVisible(t, options.First())
		require.NoError(t, field.Press("Enter"))
		again := "again :" + list[0].(string) + ": "
		waitValue(t, field, again, "Enter did not put the first name in")
		require.NoError(t, field.Press("ControlOrMeta+z"))
		if groupsTyping(engine) {
			waitUndoneToTyped(t, field, typedAlongTheWay("again :smi", "again :smi"),
				"undo did not take the completion back to a text typed before it")
		} else {
			waitValue(t, field, "again :smi", "undo did not take the completion back")
		}
		require.NoError(t, field.Press("ControlOrMeta+Shift+z"))
		waitValue(t, field, again, "redo did not put the completion back")
	})
}
