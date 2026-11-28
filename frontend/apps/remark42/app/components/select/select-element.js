import { classes } from '../../common/dom.js';
import { setAttributes } from '../../common/element.js';
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
 * @property {SelectItem} [selected]  the item shown and selected; the first one when not given
 * @property {string} [title]
 * @property {Record<string, import('../../common/element.js').AttributeValue>} [attributes]  set on
 *   the native select before its class, as the preact Select spreads the props it is given
 * @property {(value: string) => void} [onChange]  called with the value of the option picked
 */

const ICON_SIZE = { sm: 10, md: 12 };

/**
 * A select drawn as the preact Select draws it: the selected item's label over a native select
 * that holds every item. A pick the reader makes stays until the selected item passed changes, as
 * with the preact one, so with none passed the label stays the first item's after a change.
 *
 * @param {SelectProps} initial
 */
export function selectElement(initial) {
  let props = initial;
  let focused = false;
  /** @type {string | undefined} */
  let drawnItems;
  /** @type {string | number | undefined} */
  let drawnSelected;

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
    const { items, title, attributes } = props;
    const selected = props.selected ?? items[0];

    root.className = classes(
      'select',
      styles.root,
      styles[size],
      focused && styles.rootFocused,
      focused && 'select_focused',
      `select_${size}`
    );
    label.data = String(selected.label);
    if (attributes) {
      setAttributes(select, attributes);
    }
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
      drawnSelected = selected.value;
      select.replaceChildren(
        ...items.map((item) => {
          const option = document.createElement('option');
          if (item.value !== undefined) {
            option.value = String(item.value);
          }
          option.textContent = String(item.label);
          option.selected = item.value === selected.value;
          return option;
        })
      );
    } else if (selected.value !== drawnSelected) {
      drawnSelected = selected.value;
      items.forEach((item, i) => {
        select.options[i].selected = item.value === selected.value;
      });
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
