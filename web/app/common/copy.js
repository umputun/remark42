// based on https://github.com/sindresorhus/copy-text-to-clipboard, but improved to copy text styles too
module.exports = input => {
  const el = document.createElement('div');

  el.innerHTML = input;

  el.style.contain = 'strict';
  el.style.position = 'absolute';
  el.style.left = '-9999px';
  el.style.fontSize = '12pt'; // Prevent zooming on iOS

  document.body.appendChild(el);

  const currentSelection = document.getSelection();
  let originalRange = false;
  if (currentSelection.rangeCount > 0) {
    originalRange = currentSelection.getRangeAt(0);
  }

  let range, selection;

  if (document.body.createTextRange) {
    range = document.body.createTextRange();
    range.moveToElement(el);
    range.select();
  } else if (window.getSelection) {
    selection = window.getSelection();

    range = document.createRange();
    range.selectNodeContents(el);

    selection.removeAllRanges();
    selection.addRange(range);
  }

  document.execCommand('copy');

  let success = false;
  try {
    success = document.execCommand('copy');
  } catch (err) {}

  if (!document.body.createTextRange && window.getSelection) {
    window.getSelection().removeAllRanges();
  }

  document.body.removeChild(el);

  if (originalRange) {
    selection.removeAllRanges();
    selection.addRange(originalRange);
  }

  return success;
};
