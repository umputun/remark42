import clsx from 'clsx';
import { h, JSX } from 'preact';

import { BASE_URL } from 'common/constants.config';

import ghostIconUrl from './assets/ghost.svg';
import styles from './avatar.module.css';

type Props = JSX.HTMLAttributes<HTMLImageElement>;

export function Avatar({ src, title }: Props) {
  const avatarUrl = src || `${BASE_URL}${ghostIconUrl}`;

  return (
    <img
      className={clsx('avatar', styles.avatar, !src && styles.avatarGhost)}
      src={avatarUrl}
      title={title}
      aria-hidden="true"
    />
  );
}
