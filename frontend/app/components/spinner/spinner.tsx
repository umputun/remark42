import clsx from 'clsx';
import { h } from 'preact';
import { messages } from 'components/auth/auth.messsages';
import { useIntl } from 'react-intl';

import styles from './spinner.module.css';

type Props = {
  color?: 'white' | 'gray';
};

export function Spinner({ color }: Props) {
  const intl = useIntl();

  return (
    <div
      className={clsx('spinner', styles.root, { [styles.dark]: color === 'gray' })}
      role="presentation"
      aria-label={intl.formatMessage(messages.loading)}
    />
  );
}
