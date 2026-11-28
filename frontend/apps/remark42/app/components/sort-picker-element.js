// The "Sort by" control above the comments, in plain JavaScript.

import { defineMessages } from '../common/intl-message.ts';
import { selectElement } from './select/select-element.js';

/** the orders in the order they are offered */
const SORTS = /** @type {const} */ ([
  ['-score', 'best'],
  ['+score', 'worst'],
  ['-time', 'newest'],
  ['+time', 'oldest'],
  ['-active', 'recentlyUpdated'],
  ['+active', 'leastRecentlyUpdated'],
  ['-controversy', 'mostControversial'],
  ['+controversy', 'leastControversial'],
]);

/**
 * @typedef {object} SortPickerProps
 * @property {import('../common/types.ts').Sorting | undefined} sort  the order the comments are in
 * @property {(sort: import('../common/types.ts').Sorting) => void} onChange  the reader picked an order
 */

/**
 * @param {HTMLElement} root  an empty element; its class and children are this component's
 * @param {SortPickerProps} initial
 * @param {import('../common/intl-message.ts').IntlShape} initialIntl
 */
export function mountSortPicker(root, initial, initialIntl) {
  let props = initial;
  let intl = initialIntl;

  root.className = 'sort-picker';
  const label = document.createTextNode('');
  const select = selectElement(selectProps());
  root.append(label, document.createTextNode(' '), select.root);

  function selectProps() {
    const items = SORTS.map(([value, name]) => ({ value, label: intl.formatMessage(messages[name]) }));
    const sortBy = intl.formatMessage(messages.sortBy);
    return {
      items,
      selected: items.find((item) => item.value === props.sort) ?? items[0],
      attributes: { 'aria-label': sortBy },
      onChange: (/** @type {string} */ value) => {
        const known = SORTS.find(([sort]) => sort === value);
        if (known) {
          props.onChange(known[0]);
        }
      },
    };
  }

  function render() {
    label.data = intl.formatMessage(messages.sortBy);
    select.update(selectProps());
  }

  render();

  return {
    /**
     * @param {SortPickerProps} next
     * @param {import('../common/intl-message.ts').IntlShape} [nextIntl]
     */
    update(next, nextIntl = intl) {
      props = next;
      intl = nextIntl;
      render();
    },
  };
}

const messages = defineMessages({
  sortBy: {
    id: 'sort-by',
    defaultMessage: 'Sort by',
  },
  best: {
    id: 'commentsSort.best',
    defaultMessage: 'Best',
  },
  worst: {
    id: 'commentsSort.worst',
    defaultMessage: 'Worst',
  },
  newest: {
    id: 'commentsSort.newest',
    defaultMessage: 'Newest',
  },
  oldest: {
    id: 'commentsSort.oldest',
    defaultMessage: 'Oldest',
  },
  recentlyUpdated: {
    id: 'commentsSort.recently-updated',
    defaultMessage: 'Recently updated',
  },
  leastRecentlyUpdated: {
    id: 'commentsSort.least-recently-updated',
    defaultMessage: 'Least recently updated',
  },
  mostControversial: {
    id: 'commentsSort.most-controversial',
    defaultMessage: 'Most controversial',
  },
  leastControversial: {
    id: 'commentsSort.least-controversial',
    defaultMessage: 'Least controversial',
  },
});
