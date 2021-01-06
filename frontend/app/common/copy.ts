// based on https://github.com/sindresorhus/copy-text-to-clipboard, but improved to copy text styles too
export default function copy(input: string): boolean {
  const element = document.createElement('textarea') as HTMLTextAreaElement;
  const previouslyFocusedElement = document.activeElement as HTMLElement;

  element.value = input;

  // Prevent keyboard from showing on mobile
  element.setAttribute('readonly', '');

  Object.assign(element.style, {
    contain: 'strict',
    position: 'absolute',
    left: '-9999px',
    fontSize: '12pt', // Prevent zooming on iOS
  });

  const selection = document.getSelection();
  let originalRange: boolean | Range = false;

  if (selection && selection.rangeCount > 0) {
    originalRange = selection.getRangeAt(0);
  }

  document.body.append(element);
  element.select();

  // Explicit selection workaround for iOS
  element.selectionStart = 0;
  element.selectionEnd = input.length;

  let isSuccess = false;
  try {
    isSuccess = document.execCommand('copy');
  } catch (_) {}

  element.remove();

  if (selection && originalRange) {
    selection.removeAllRanges();
    selection.addRange(originalRange);
  }

  // Get the focus back on the previously focused element, if any
  if (previouslyFocusedElement) {
    previouslyFocusedElement.focus();
  }

  return isSuccess;
}
