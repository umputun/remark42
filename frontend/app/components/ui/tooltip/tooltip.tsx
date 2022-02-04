import clsx from 'clsx';
import { h, ComponentChildren } from 'preact';

import styles from './tooltip.module.css';

type Props = {
  children?: ComponentChildren;
  text: string;
  position?: 'bottom-left';
};

export function Tooltip({ children, text, position }: Props) {
  return (
    <div className={clsx('', styles.root)}>
      {children}
      <div className={clsx('tooltip', styles.tooltip, position && styles[`tooltip_${position}`])}>{text}</div>
    </div>
  );
}
