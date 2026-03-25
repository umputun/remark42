import clsx from 'clsx';
import { h, JSX } from 'preact';
import { forwardRef } from 'preact/compat';

import type { Theme } from 'common/types';

import styles from './button.module.css';

const kindStyles: Record<string, string> = {
  primary: styles.kindPrimary,
  secondary: styles.kindSecondary,
  link: styles.kindLink,
};

const sizeStyles: Record<string, string> = {
  middle: styles.sizeMiddle,
  large: styles.sizeLarge,
};

export type ButtonProps = Omit<JSX.HTMLAttributes, 'size' | 'className'> & {
  kind?: 'primary' | 'secondary' | 'link';
  size?: 'middle' | 'large';
  theme?: Theme;
  mix?: string | string[];
  type?: string;
  className?: string;
};

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  ({ children, theme, mix, kind, type = 'button', size, className, ...props }, ref) => (
    <button
      className={clsx(
        styles.root,
        kind && kindStyles[kind],
        size && sizeStyles[size],
        theme === 'dark' && styles.themeDark,
        mix,
        className
      )}
      type={type}
      {...props}
      ref={ref}
    >
      {children}
    </button>
  )
);
