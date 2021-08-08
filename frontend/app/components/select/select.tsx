import clsx from 'clsx';
import { Arrow } from 'components/icons/arrow';
import { h, JSX } from 'preact';
import { useState } from 'preact/hooks';
import styles from './select.module.css';

type Item = {
  label: string | number;
  value: string | number;
};

type Props = {
  items: Item[];
  selected: Item;
  onChange?: JSX.GenericEventHandler<HTMLSelectElement>;
};

export function Select({ items, selected, onChange }: Props) {
  const [focus, setFocus] = useState(false);

  return (
    <span className={clsx('select', styles.root, focus && styles.rootFocused)}>
      {selected.label}
      <Arrow className={clsx('select-arrow', styles.arrow)} />
      <select
        onFocus={() => setFocus(true)}
        onBlur={() => setFocus(false)}
        className={clsx('select-element', styles.select)}
        onChange={onChange}
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
