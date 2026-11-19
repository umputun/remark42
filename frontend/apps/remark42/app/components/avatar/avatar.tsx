import clsx from 'clsx';
import { h } from 'preact';

import { webAssetUrl } from 'common/web-asset';

import styles from './avatar.module.css';

type Props = {
  url?: string;
  /** the picture is a generated identicon, which scales without blurring its blocks only pixelated */
  identicon?: boolean;
};

export function Avatar({ url, identicon }: Props) {
  const avatarUrl = url || webAssetUrl('ghost.svg');

  return (
    // eslint-disable-next-line jsx-a11y/alt-text
    <img
      className={clsx('avatar', styles.avatar, !url && styles.avatarGhost, url && identicon && styles.avatarIdenticon)}
      src={avatarUrl}
      aria-hidden="true"
    />
  );
}
