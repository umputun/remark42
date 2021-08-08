import clsx from 'clsx';
import { h, JSX } from 'preact';
import { useState } from 'preact/hooks';

import { ArrowIcon } from 'components/icons/arrow';

import styles from './select.module.css';

type Item = {
  label: string | number;
  value: string | number;
};

type Props = {
  items: Item[];
  selected: Item;
} & Omit<JSX.HTMLAttributes<HTMLSelectElement>, 'className' | 'onFocus' | 'onBlur' | 'selected'>;

export function Select({ items, selected, ...props }: Props) {
  const [focus, setFocus] = useState(false);

  return (
    <span className={clsx('select', styles.root, { [styles.rootFocused]: focus, select_focused: focus })}>
      {selected.label}
      <ArrowIcon className={clsx('select-arrow', styles.arrow)} />
      <select
        {...props}
        onFocus={() => setFocus(true)}
        onBlur={() => setFocus(false)}
        className={clsx('select-element', styles.select)}
      >
        {items.map((i) => (
          <option key={i.value} value={i.value} selected={selected.value === i.value}>
            {i.label}
          </option>
        ))}
      </select>
    </span>
  );
}
