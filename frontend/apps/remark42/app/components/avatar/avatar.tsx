import clsx from 'clsx';
import { h } from 'preact';

import { webAssetUrl } from 'common/web-asset';

import styles from './avatar.module.css';

type Props = {
  url?: string;
  /** the side of the square the picture is shown at, in CSS pixels, which its wrapper also sets */
  size: number;
  /** the picture is a generated identicon, which scales without blurring its blocks only pixelated */
  identicon?: boolean;
};

export function Avatar({ url, size, identicon }: Props) {
  const avatarUrl = url || webAssetUrl('ghost.svg');

  return (
    <img
      className={clsx('avatar', styles.avatar, !url && styles.avatarGhost, url && identicon && styles.avatarIdenticon)}
      src={avatarUrl}
      width={size}
      height={size}
      alt=""
      decoding="async"
      aria-hidden="true"
    />
  );
}
