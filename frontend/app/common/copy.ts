export function copy(content: string): boolean {
  // We use `div` instead of `input` or `textarea` because we want to copy styles
  const container = document.createElement('div');
  const previouslyFocusedElement = document.activeElement as HTMLElement;

  container.innerHTML = content;

  Object.assign(container.style, {
    contain: 'strict',
    position: 'absolute',
    left: '-9999px',
    fontSize: '12pt', // Prevent zooming on iOS
  });

  document.body.appendChild(container);

  let selection = window.getSelection();
  // save original selection
  const originalRange = selection && selection.rangeCount > 0 ? selection.getRangeAt(0) : null;
  const range = document.createRange();

  range.selectNodeContents(container);

  if (selection) {
    selection.removeAllRanges();
    selection.addRange(range);
  }

  document.execCommand('copy');

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

  return success;
}
