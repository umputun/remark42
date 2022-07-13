import clsx from 'clsx';
import { h } from 'preact';

import { BASE_URL } from 'common/constants.config';

import ghostIconUrl from './assets/ghost.svg';
import styles from './avatar.module.css';

type Props = {
  url?: string;
};

export function Avatar({ url }: Props) {
  const avatarUrl = url || `${BASE_URL}${ghostIconUrl}`;

  return (
    // eslint-disable-next-line jsx-a11y/alt-text
    <img className={clsx('avatar', styles.avatar, !url && styles.avatarGhost)} src={avatarUrl} aria-hidden="true" />
  );
}
