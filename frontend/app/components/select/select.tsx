import clsx from 'clsx';
import { h, JSX } from 'preact';
import { useState } from 'preact/hooks';

import { ArrowIcon } from 'components/icons/arrow';

import styles from './select.module.css';

type Item = {
  label: string | number;
  value: string | number | undefined;
};

type Props = Omit<
  JSX.HTMLAttributes<HTMLSelectElement>,
  'className' | 'onFocus' | 'onBlur' | 'selected' | 'label' | 'icon' | 'size'
> & {
  size?: 'sm' | 'md';
  items: Item[];
  selected?: Item;
};

export function Select({ items, selected, size = 'md', ...props }: Props) {
  const [focus, setFocus] = useState(false);
  const selectedItem = selected ?? items[0];

  const iconSize = {
    sm: 10,
    md: 12,
  };

  return (
    <span
      data-testid="select-root"
      className={clsx('select', styles.root, size && styles[size], {
        [styles.rootFocused]: focus,
        select_focused: focus,
        [`select_${size}`]: size,
      })}
    >
      {selectedItem.label}
      <ArrowIcon data-testid="select-arrow" size={iconSize[size]} className={clsx('select-arrow', styles.arrow)} />
      <select
        {...props}
        onFocus={() => setFocus(true)}
        onBlur={() => setFocus(false)}
        className={clsx('select-element', styles.select)}
        // wrong typings in preact lib
        // @ts-ignore
        selected={selectedItem.value}
      >
        {items.map((i) => (
          <option key={i.value} value={i.value}>
            {i.label}
          </option>
        ))}
      </select>
    </span>
  );
}
