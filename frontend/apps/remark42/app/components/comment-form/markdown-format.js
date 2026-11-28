/*!
 * Remark42's markdown toolbar edits are adapted from @github/markdown-toolbar-element 2.2.3,
 * under its licence:
 *
 * Copyright (c) 2017-2018 GitHub, Inc.
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

// What each markdown toolbar button does to the text and the selection, as the toolbar package
// did it for the eight buttons the form has. Nothing here touches the page: the toolbar reads the
// field's text and selection, asks for the edit, and makes it.

/**
 * @typedef {object} Style
 * @property {string} [prefix]
 * @property {string} [suffix]
 * @property {string} [blockPrefix]  used in place of the prefix around a selection of several lines
 * @property {string} [blockSuffix]
 * @property {boolean} [multiline]  each selected line gets the prefix and suffix
 * @property {string} [replaceNext]  text in the suffix that is selected after the edit
 * @property {string} [scanFor]  a selection matching this goes in place of replaceNext
 * @property {boolean} [surroundWithNewlines]  the result stands in a paragraph of its own
 * @property {boolean} [orderedList]
 * @property {boolean} [unorderedList]
 * @property {boolean} [trimFirst]  whitespace at the edges of the selection stays outside the markers
 */

/** @type {Record<string, Style>} */
export const STYLES = {
  // the toolbar's heading button has no level set, which makes it a third-level heading
  header: { prefix: '### ' },
  bold: { prefix: '**', suffix: '**', trimFirst: true },
  italic: { prefix: '_', suffix: '_', trimFirst: true },
  quote: { prefix: '> ', multiline: true, surroundWithNewlines: true },
  code: { prefix: '`', suffix: '`', blockPrefix: '```', blockSuffix: '```' },
  link: { prefix: '[', suffix: '](url)', replaceNext: 'url', scanFor: 'https?://' },
  'unordered-list': { prefix: '- ', multiline: true, unorderedList: true },
  'ordered-list': { prefix: '1. ', multiline: true, orderedList: true },
};

/**
 * @typedef {object} Edit
 * @property {number} start  the text from start to end is replaced
 * @property {number} end
 * @property {string} text  what replaces it
 * @property {number} selectionStart  the selection once it is replaced
 * @property {number} selectionEnd
 */

/**
 * The field the edit is worked out on: its text, and a selection that moves as the package moved
 * the field's own while it looked at the text around it.
 *
 * @typedef {object} Field
 * @property {string} value
 * @property {number} selectionStart
 * @property {number} selectionEnd
 */

/** @param {string} text */
function isMultipleLines(text) {
  return text.trim().split('\n').length > 1;
}

/**
 * @param {string} text
 * @param {number} i
 */
function wordSelectionStart(text, i) {
  let index = i;
  while (text[index] && text[index - 1] != null && !text[index - 1].match(/\s/)) {
    index--;
  }
  return index;
}

/**
 * @param {string} text
 * @param {number} i
 * @param {boolean} multiline
 */
function wordSelectionEnd(text, i, multiline) {
  let index = i;
  const breakpoint = multiline ? /\n/ : /\s/;
  while (text[index] && !text[index].match(breakpoint)) {
    index++;
  }
  return index;
}

/** @param {Field} field */
function expandSelectionToLine(field) {
  const lines = field.value.split('\n');
  let counter = 0;
  for (const line of lines) {
    const lineLength = line.length + 1;
    if (field.selectionStart >= counter && field.selectionStart < counter + lineLength) {
      field.selectionStart = counter;
    }
    if (field.selectionEnd >= counter && field.selectionEnd < counter + lineLength) {
      field.selectionEnd = counter + lineLength - 1;
    }
    counter += lineLength;
  }
}

/**
 * @param {Field} field
 * @param {string} prefix
 * @param {string} suffix
 * @param {boolean} multiline
 */
function expandSelectedText(field, prefix, suffix, multiline) {
  if (field.selectionStart === field.selectionEnd) {
    field.selectionStart = wordSelectionStart(field.value, field.selectionStart);
    field.selectionEnd = wordSelectionEnd(field.value, field.selectionEnd, multiline);
  } else {
    const expandedStart = field.selectionStart - prefix.length;
    const expandedEnd = field.selectionEnd + suffix.length;
    const beginsWithPrefix = field.value.slice(expandedStart, field.selectionStart) === prefix;
    const endsWithSuffix = field.value.slice(field.selectionEnd, expandedEnd) === suffix;
    if (beginsWithPrefix && endsWithSuffix) {
      field.selectionStart = expandedStart;
      field.selectionEnd = expandedEnd;
    }
  }
  return field.value.slice(field.selectionStart, field.selectionEnd);
}

/** @param {Field} field */
function newlinesToSurroundSelectedText(field) {
  const before = field.value.slice(0, field.selectionStart);
  const after = field.value.slice(field.selectionEnd);
  const breaksBefore = before.match(/\n*$/);
  const breaksAfter = after.match(/^\n*/);
  const newlinesBefore = breaksBefore ? breaksBefore[0].length : 0;
  const newlinesAfter = breaksAfter ? breaksAfter[0].length : 0;
  const newlinesToAppend = before.match(/\S/) && newlinesBefore < 2 ? '\n'.repeat(2 - newlinesBefore) : '';
  const newlinesToPrepend = after.match(/\S/) && newlinesAfter < 2 ? '\n'.repeat(2 - newlinesAfter) : '';
  return { newlinesToAppend, newlinesToPrepend };
}

/**
 * @param {Field} field
 * @param {Required<Style>} style
 * @returns {{ text: string, selectionStart: number, selectionEnd: number }}
 */
function blockStyle(field, style) {
  const { prefix, suffix, blockPrefix, blockSuffix, replaceNext, scanFor, surroundWithNewlines } = style;
  const originalStart = field.selectionStart;
  const originalEnd = field.selectionEnd;
  let selectedText = field.value.slice(field.selectionStart, field.selectionEnd);
  let prefixToUse = isMultipleLines(selectedText) && blockPrefix.length > 0 ? `${blockPrefix}\n` : prefix;
  let suffixToUse = isMultipleLines(selectedText) && blockSuffix.length > 0 ? `\n${blockSuffix}` : suffix;

  selectedText = expandSelectedText(field, prefixToUse, suffixToUse, style.multiline);
  let selectionStart = field.selectionStart;
  let selectionEnd = field.selectionEnd;
  const hasReplaceNext = replaceNext.length > 0 && suffixToUse.indexOf(replaceNext) > -1 && selectedText.length > 0;
  if (surroundWithNewlines) {
    const { newlinesToAppend, newlinesToPrepend } = newlinesToSurroundSelectedText(field);
    prefixToUse = newlinesToAppend + prefix;
    suffixToUse += newlinesToPrepend;
  }

  if (selectedText.startsWith(prefixToUse) && selectedText.endsWith(suffixToUse)) {
    const replacementText = selectedText.slice(prefixToUse.length, selectedText.length - suffixToUse.length);
    if (originalStart === originalEnd) {
      let position = originalStart - prefixToUse.length;
      position = Math.max(position, selectionStart);
      position = Math.min(position, selectionStart + replacementText.length);
      selectionStart = selectionEnd = position;
    } else {
      selectionEnd = selectionStart + replacementText.length;
    }
    return { text: replacementText, selectionStart, selectionEnd };
  }
  if (!hasReplaceNext) {
    let replacementText = prefixToUse + selectedText + suffixToUse;
    selectionStart = originalStart + prefixToUse.length;
    selectionEnd = originalEnd + prefixToUse.length;
    const whitespaceEdges = selectedText.match(/^\s*|\s*$/g);
    if (style.trimFirst && whitespaceEdges) {
      const leadingWhitespace = whitespaceEdges[0] || '';
      const trailingWhitespace = whitespaceEdges[1] || '';
      replacementText = leadingWhitespace + prefixToUse + selectedText.trim() + suffixToUse + trailingWhitespace;
      selectionStart += leadingWhitespace.length;
      selectionEnd -= trailingWhitespace.length;
    }
    return { text: replacementText, selectionStart, selectionEnd };
  }
  if (scanFor.length > 0 && selectedText.match(scanFor)) {
    suffixToUse = suffixToUse.replace(replaceNext, selectedText);
    const replacementText = prefixToUse + suffixToUse;
    selectionStart = selectionEnd = selectionStart + prefixToUse.length;
    return { text: replacementText, selectionStart, selectionEnd };
  }
  const replacementText = prefixToUse + selectedText + suffixToUse;
  selectionStart = selectionStart + prefixToUse.length + selectedText.length + suffixToUse.indexOf(replaceNext);
  selectionEnd = selectionStart + replaceNext.length;
  return { text: replacementText, selectionStart, selectionEnd };
}

/**
 * @param {Field} field
 * @param {Required<Style>} style
 */
function multilineStyle(field, style) {
  const { prefix, suffix, surroundWithNewlines } = style;
  let text = field.value.slice(field.selectionStart, field.selectionEnd);
  let selectionStart = field.selectionStart;
  let selectionEnd = field.selectionEnd;
  const lines = text.split('\n');
  const undoStyle = lines.every((line) => line.startsWith(prefix) && line.endsWith(suffix));
  if (undoStyle) {
    text = lines.map((line) => line.slice(prefix.length, line.length - suffix.length)).join('\n');
    selectionEnd = selectionStart + text.length;
  } else {
    text = lines.map((line) => prefix + line + suffix).join('\n');
    if (surroundWithNewlines) {
      const { newlinesToAppend, newlinesToPrepend } = newlinesToSurroundSelectedText(field);
      selectionStart += newlinesToAppend.length;
      selectionEnd = selectionStart + text.length;
      text = newlinesToAppend + text + newlinesToPrepend;
    }
  }
  return { text, selectionStart, selectionEnd };
}

/** @param {string} text */
function undoOrderedListStyle(text) {
  const lines = text.split('\n');
  const orderedListRegex = /^\d+\.\s+/;
  const processed = lines.every((line) => orderedListRegex.test(line));
  const result = processed ? lines.map((line) => line.replace(orderedListRegex, '')) : lines;
  return { text: result.join('\n'), processed };
}

/** @param {string} text */
function undoUnorderedListStyle(text) {
  const lines = text.split('\n');
  const processed = lines.every((line) => line.startsWith('- '));
  const result = processed ? lines.map((line) => line.slice(2)) : lines;
  return { text: result.join('\n'), processed };
}

/**
 * @param {number} index
 * @param {boolean} unorderedList
 */
function makePrefix(index, unorderedList) {
  return unorderedList ? '- ' : `${index + 1}. `;
}

/**
 * @param {Field} field
 * @param {Required<Style>} style
 */
function listStyle(field, style) {
  const noInitialSelection = field.selectionStart === field.selectionEnd;
  let selectionStart = field.selectionStart;
  let selectionEnd = field.selectionEnd;
  expandSelectionToLine(field);
  const selectedText = field.value.slice(field.selectionStart, field.selectionEnd);
  const undoResult = style.orderedList ? undoOrderedListStyle(selectedText) : undoUnorderedListStyle(selectedText);
  const undoOppositeList = style.orderedList
    ? undoUnorderedListStyle(undoResult.text)
    : undoOrderedListStyle(undoResult.text);
  const pristineText = undoOppositeList.text;
  const prefixedLines = pristineText
    .split('\n')
    .map((value, index) => `${makePrefix(index, style.unorderedList)}${value}`);
  const totalPrefixLength = prefixedLines.reduce(
    (total, _line, index) => total + makePrefix(index, style.unorderedList).length,
    0
  );
  const totalPrefixLengthOppositeList = prefixedLines.reduce(
    (total, _line, index) => total + makePrefix(index, !style.unorderedList).length,
    0
  );

  if (undoResult.processed) {
    if (noInitialSelection) {
      selectionStart = Math.max(selectionStart - makePrefix(0, style.unorderedList).length, 0);
      selectionEnd = selectionStart;
    } else {
      selectionStart = field.selectionStart;
      selectionEnd = field.selectionEnd - totalPrefixLength;
    }
    return { text: pristineText, selectionStart, selectionEnd };
  }

  const { newlinesToAppend, newlinesToPrepend } = newlinesToSurroundSelectedText(field);
  const text = newlinesToAppend + prefixedLines.join('\n') + newlinesToPrepend;
  if (noInitialSelection) {
    selectionStart = Math.max(selectionStart + makePrefix(0, style.unorderedList).length + newlinesToAppend.length, 0);
    selectionEnd = selectionStart;
  } else if (undoOppositeList.processed) {
    selectionStart = Math.max(field.selectionStart + newlinesToAppend.length, 0);
    selectionEnd = field.selectionEnd + newlinesToAppend.length + totalPrefixLength - totalPrefixLengthOppositeList;
  } else {
    selectionStart = Math.max(field.selectionStart + newlinesToAppend.length, 0);
    selectionEnd = field.selectionEnd + newlinesToAppend.length + totalPrefixLength;
  }
  return { text, selectionStart, selectionEnd };
}

/**
 * The edit a button makes to this text and selection.
 *
 * @param {string} value
 * @param {number} selectionStart
 * @param {number} selectionEnd
 * @param {Style} style
 * @returns {Edit}
 */
export function formatSelection(value, selectionStart, selectionEnd, style) {
  /** @type {Required<Style>} */
  const full = {
    prefix: '',
    suffix: '',
    blockPrefix: '',
    blockSuffix: '',
    multiline: false,
    replaceNext: '',
    scanFor: '',
    surroundWithNewlines: false,
    orderedList: false,
    unorderedList: false,
    trimFirst: false,
    ...style,
  };
  /** @type {Field} */
  const field = { value, selectionStart, selectionEnd };
  const selected = value.slice(selectionStart, selectionEnd);
  let result;
  if (full.orderedList || full.unorderedList) {
    result = listStyle(field, full);
  } else if (full.multiline && isMultipleLines(selected)) {
    result = multilineStyle(field, full);
  } else {
    result = blockStyle(field, full);
  }
  return { start: field.selectionStart, end: field.selectionEnd, ...result };
}
