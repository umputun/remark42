import clsx from 'clsx';
import { h } from 'preact';
import { messages } from 'components/auth/auth.messsages';
import { useIntl } from 'react-intl';

import styles from './spinner.module.css';

export function Spinner() {
  const intl = useIntl();

  return (
    <div
      className={clsx('spinner', styles.root)}
      role="presentation"
      aria-label={intl.formatMessage(messages.loading)}
    />
  );
}
