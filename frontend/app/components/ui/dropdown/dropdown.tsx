import clsx from 'clsx';
import { cloneElement, ComponentChildren, h, VNode } from 'preact';

import { useDropdown } from './dropdown.hooks';

import styles from './dropdown.module.css';

type Props = {
  button: VNode;
  position?: 'bottom-left' | 'bottom-right';
  children: ComponentChildren;
};

export function Dropdown({ button, position, children }: Props) {
  const [rootRef, isDropdownShowed, toggleDropdownState] = useDropdown(false);

  return (
    <div className={styles.root}>
      {cloneElement(button, { onClick: toggleDropdownState })}
      {isDropdownShowed && (
        // TODO: add static class `dropdown` when old dropdown is removed
        <div ref={rootRef} className={clsx(styles.dropdown, position && styles[`dropdown_${position}`])}>
          {children}
        </div>
      )}
    </div>
  );
}
