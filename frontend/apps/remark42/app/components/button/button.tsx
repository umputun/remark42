import { h, JSX } from 'preact';
import { forwardRef } from 'preact/compat';
import b, { Mods, Mix } from 'bem-react-helper';

import type { Theme } from 'common/types';

export type ButtonProps = Omit<JSX.HTMLAttributes, 'size' | 'className'> & {
  kind?: 'primary' | 'secondary' | 'link';
  size?: 'middle' | 'large';
  theme?: Theme;
  mods?: Mods;
  mix?: Mix;
  type?: string;
};

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  ({ children, theme, mods, mix, kind, type = 'button', size, ...props }, ref) => (
    <button className={b('button', { mods: { kind, size, theme }, mix }, { ...mods })} type={type} {...props} ref={ref}>
      {children}
    </button>
  )
);
