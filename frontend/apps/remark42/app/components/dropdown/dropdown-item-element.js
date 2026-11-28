import styles from './__item/dropdown-item.module.css';

/**
 * One row of a dropdown, drawn as the preact DropdownItem draws it, holding the given nodes.
 *
 * @param {Node[]} children
 */
export function dropdownItemElement(children) {
  const item = document.createElement('div');
  item.className = styles.root;
  item.append(...children);
  return item;
}
