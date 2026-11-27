import { classes } from '../../common/dom.js';
import { arrowIcon } from '../icons/arrow-icon.js';
import styles from './select.module.css';

/**
 * @typedef {object} SelectItem
 * @property {string | number} label
 * @property {string | number | undefined} value
 */

/**
 * @typedef {object} SelectProps
 * @property {SelectItem[]} items
 * @property {'sm' | 'md'} [size]
 * @property {string} [title]
 * @property {(value: string) => void} [onChange]  called with the value of the option picked
 */

const ICON_SIZE = { sm: 10, md: 12 };

/**
 * A select drawn as the preact Select draws it: the first item's label over a native select
 * that holds every item. The label stays the first item's after a change, as the preact one's
 * does when no item is passed as selected.
 *
 * @param {SelectProps} initial
 */
export function selectElement(initial) {
  let props = initial;
  let focused = false;
  /** @type {string | undefined} */
  let drawnItems;

  const root = document.createElement('span');
  root.dataset.testid = 'select-root';
  const label = document.createTextNode('');
  const size = props.size ?? 'md';
  const arrow = arrowIcon(ICON_SIZE[size], {
    'data-testid': 'select-arrow',
    class: classes('select-arrow', styles.arrow),
  });
  const select = document.createElement('select');
  select.addEventListener('focus', () => {
    focused = true;
    render();
  });
  select.addEventListener('blur', () => {
    focused = false;
    render();
  });
  select.addEventListener('change', () => props.onChange?.(select.value));
  root.append(label, arrow, select);
  render();

  function render() {
    const { items, title } = props;

    root.className = classes(
      'select',
      styles.root,
      styles[size],
      focused && styles.rootFocused,
      focused && 'select_focused',
      `select_${size}`
    );
    label.data = String(items[0].label);
    if (title === undefined) {
      select.removeAttribute('title');
    } else {
      select.title = title;
    }
    select.className = classes('select-element', styles.select);

    // the options are built again only when the items change, so a pick the reader made stays
    const key = JSON.stringify(items);
    if (key !== drawnItems) {
      drawnItems = key;
      select.replaceChildren(
        ...items.map((item, i) => {
          const option = document.createElement('option');
          if (item.value !== undefined) {
            option.value = String(item.value);
          }
          option.textContent = String(item.label);
          option.selected = i === 0;
          return option;
        })
      );
    }
  }

  return {
    root,
    /** @param {SelectProps} next */
    update(next) {
      props = next;
      render();
    },
  };
}
