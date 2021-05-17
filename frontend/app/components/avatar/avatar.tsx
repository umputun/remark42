import clsx from 'clsx';
import { h } from 'preact';

import { BASE_URL } from 'common/constants.config';

import styles from './avatar.module.css';

type Props = {
  url?: string;
  alt: string;
  /** className should be used only in puprose of put permanent class on Avatar for user themization */
  className: string;
};

export function Avatar({ url, className, alt }: Props) {
  const avatarUrl = url || `${BASE_URL}${require('./assets/ghost.svg').default}`;

  return (
    // eslint-disable-next-line jsx-a11y/alt-text
    <img className={clsx('avatar', className, styles.avatar, !url && styles.avatarGhost)} src={avatarUrl} alt={alt} />
  );
}
