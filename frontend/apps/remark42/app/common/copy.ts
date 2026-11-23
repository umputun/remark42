export async function copy(content: string): Promise<void> {
  // Use new API for modern browsers
  if ('clipboard' in window.navigator) {
    const blob = new Blob([content], { type: 'text/plain' });
    return navigator.clipboard.write([new ClipboardItem({ [blob.type]: blob })]);
  }

  // No clipboard API: browsers expose it in a secure context only, so a page served over plain
  // http has none unless it comes from localhost or a loopback address. The text goes into an
  // off-screen element that is selected and copied
  const container = document.createElement('div');
  const previouslyFocusedElement = document.activeElement as HTMLElement;

  // `content` is plain text, as in the branch above. Parsed as HTML, markup a commenter wrote as
  // text would become elements
  container.textContent = content;

  Object.assign(container.style, {
    contain: 'strict',
    position: 'absolute',
    left: '-9999px',
    fontSize: '12pt', // Prevent zooming on iOS
    whiteSpace: 'pre', // Keep the line breaks of the text in the copy
  });

  document.body.appendChild(container);

  const selection = window.getSelection();
  // save original selection
  const originalRange = selection && selection.rangeCount > 0 ? selection.getRangeAt(0) : null;
  const range = document.createRange();

  range.selectNodeContents(container);

  if (selection) {
    selection.removeAllRanges();
    selection.addRange(range);
  }

  let success = false;
  try {
    success = document.execCommand('copy');
  } catch (err) {}

  if (selection) {
    selection.removeAllRanges();
  }

  document.body.removeChild(container);

  // Put the selection back in case had it before
  if (originalRange && selection) {
    selection.addRange(originalRange);
  }

  // Get the focus back on the previously focused element, if any
  if (previouslyFocusedElement) {
    previouslyFocusedElement.focus();
  }

  return success ? Promise.resolve() : Promise.reject();
}
