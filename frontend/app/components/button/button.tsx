/** @jsx createElement */
import { createElement, JSX } from 'preact';
import { forwardRef } from 'preact/compat';
import b, { Mods, Mix } from 'bem-react-helper';
import { Theme } from '@app/common/types';

interface Props extends Omit<JSX.HTMLAttributes, 'size' | 'className'> {
  kind?: 'primary' | 'secondary' | 'link';
  size?: 'middle' | 'large';
  theme?: Theme;
  mods?: Mods;
  mix?: Mix;
  type?: string;
}

export const Button = forwardRef<HTMLButtonElement, Props>(
  ({ children, theme, mods, mix, kind, type = 'button', size, ...props }) => {
    const className = b('button', { mods: { kind, size }, mix }, { theme, ...mods });

    return (
      <button className={className} type={type} {...props}>
        {children}
      </button>
    );
  }
);
