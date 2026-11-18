import clsx from 'clsx';
import { h } from 'preact';

import { webAssetUrl } from 'common/web-asset';

import styles from './avatar.module.css';

type Props = {
  url?: string;
};

export function Avatar({ url }: Props) {
  const avatarUrl = url || webAssetUrl('ghost.svg');

  return (
    // eslint-disable-next-line jsx-a11y/alt-text
    <img className={clsx('avatar', styles.avatar, !url && styles.avatarGhost)} src={avatarUrl} aria-hidden="true" />
  );
}
