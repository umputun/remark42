import { h } from 'preact';
import clsx from 'clsx';

type Props = {
  className?: string;
};

export function Preloader({ className }: Props) {
  return <div className={clsx('preloader', className)} aria-label="Loading..." data-testid="preloader" />;
}
