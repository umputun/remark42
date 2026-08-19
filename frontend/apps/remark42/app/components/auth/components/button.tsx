import { h, VNode, type ButtonHTMLAttributes } from 'preact';
import clsx from 'clsx';

import styles from './button.module.css';

type Props = Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'size'> & {
  size?: 'xs' | 'sm';
  kind?: 'transparent' | 'link' | 'hollow';
  suffix?: VNode;
  loading?: boolean;
  selected?: boolean;
};

export function Button({ children, size, kind, suffix, selected, className, onChange, ...props }: Props) {
  return (
    <button
      className={clsx(className, styles.button, kind && styles[kind], size && styles[size], {
        [styles.selected]: selected,
      })}
      {...props}
    >
      {children}
      {suffix && <div className={styles.suffix}>{suffix}</div>}
    </button>
  );
}
