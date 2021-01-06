// based on https://github.com/sindresorhus/copy-text-to-clipboard, but improved to copy text styles too
export default function copy(input: string): boolean {
  const el = document.createElement('div');

  el.innerHTML = input;

  Object.assign(el.style, {
    contain: 'strict',
    position: 'absolute',
    left: '-9999px',
    fontSize: '12pt', // Prevent zooming on iOS
  });

  document.body.appendChild(el);

  const currentSelection = document.getSelection();
  if (!currentSelection) return true;

  let originalRange = null;
  if (currentSelection.rangeCount > 0) {
    originalRange = currentSelection.getRangeAt(0);
  }

  let range, selection;

  if ((document.body as any).createTextRange) {
    range = (document.body as any).createTextRange();
    range.moveToElement(el);
    range.select();
  } else if (window.getSelection) {
    selection = window.getSelection();

    range = document.createRange();
    range.selectNodeContents(el);

    if (selection) {
      selection.removeAllRanges();
      selection.addRange(range);
    }
  }

  document.execCommand('copy');

  let success = false;
  try {
    success = document.execCommand('copy');
  } catch (err) {}

  if (!(document.body as any).createTextRange && window.getSelection) {
    const selection = window.getSelection();
    selection && selection.removeAllRanges();
  }

  document.body.removeChild(el);

  if (originalRange) {
    (selection as any).removeAllRanges();
    (selection as any).addRange(originalRange);
  }

  return success;
}
