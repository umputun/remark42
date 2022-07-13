import { h } from 'preact';
import clsx from 'clsx';
import styles from './tooltip.module.css';
import { useEffect } from 'preact/hooks';

type Props = {
  children: preact.ComponentChild;
  className?: string;
  content?: preact.ComponentChild;
  permanent?: boolean;
  position: 'top-left' | 'top-right';
  hideBehavior?: 'mouseleave' | 'click';
  hideTimeout?: number;
  onHide?(): void;
};

export function Tooltip({
  className,
  children,
  content,
  permanent,
  hideBehavior,
  hideTimeout,
  position,
  onHide,
}: Props) {
  useEffect(() => {
    if (content && hideTimeout && onHide) {
      setTimeout(onHide, hideTimeout);
    }
  }, [content, hideTimeout, onHide]);

  return (
    <div className={clsx(styles.root, className)}>
      {content && (
        <div
          role="tooltip"
          className={clsx(styles.tooltip, permanent && styles.tooltipPermanent, styles[position])}
          onMouseLeave={hideBehavior === 'mouseleave' ? onHide : undefined}
        >
          {content}
        </div>
      )}
      {children}
    </div>
  );
}
