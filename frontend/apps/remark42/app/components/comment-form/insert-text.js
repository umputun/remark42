// Adapted from @github/markdown-toolbar-element 2.2.3, under the licence in markdown-format.js.

// Putting text into the comment field the way a reader's typing would, so the browser's undo takes
// it back. This is how the markdown toolbar package inserted its text, fallbacks included.

/** null until the first insert has shown whether the browser can insert into a textarea */
let canInsertText = /** @type {boolean | null} */ (null);

/**
 * Replaces the field's selected text with this text, then selects the given range.
 *
 * The browser's own insertText command keeps the edit in the undo history and fires the input
 * event itself. A browser that has no such command for a textarea, or inserts something other
 * than the text, gets the value set directly and an input event dispatched, wrapped in the undo
 * unit of the old Edge; that edit cannot be undone.
 *
 * @param {HTMLTextAreaElement} textarea
 * @param {string} text
 * @param {number} selectionStart
 * @param {number} selectionEnd
 */
export function insertText(textarea, text, selectionStart, selectionEnd) {
  const before = textarea.value.slice(0, textarea.selectionStart);
  const after = textarea.value.slice(textarea.selectionEnd);

  if (canInsertText === null || canInsertText === true) {
    // an older Firefox runs the command on a textarea only while it is editable content
    textarea.contentEditable = 'true';
    try {
      canInsertText = document.execCommand('insertText', false, text);
    } catch {
      canInsertText = false;
    }
    textarea.contentEditable = 'false';
  }

  if (canInsertText && !textarea.value.slice(0, textarea.selectionStart).endsWith(text)) {
    canInsertText = false;
  }

  if (!canInsertText) {
    try {
      document.execCommand('ms-beginUndoUnit');
    } catch {
      // only the old Edge has it
    }
    textarea.value = before + text + after;
    try {
      document.execCommand('ms-endUndoUnit');
    } catch {
      // only the old Edge has it
    }
    textarea.dispatchEvent(new CustomEvent('input', { bubbles: true, cancelable: true }));
  }

  textarea.setSelectionRange(selectionStart, selectionEnd);
}

/** Forgets what the last insert showed about the browser; for tests. */
export function resetInsertText() {
  canInsertText = null;
}
