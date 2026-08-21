import { h } from 'preact';
import { useIntl, defineMessages } from 'react-intl';

import styles from './confirm-dialog.module.css';

export type Props = {
  message: string;
  onConfirm(): void;
  onCancel(): void;
};

const messages = defineMessages({
  confirm: { id: 'confirm-dialog.confirm', defaultMessage: 'Confirm' },
  cancel: { id: 'confirm-dialog.cancel', defaultMessage: 'Cancel' },
});

export function ConfirmDialog({ message, onConfirm, onCancel }: Props) {
  const intl = useIntl();

  return (
    <div className={styles.root} role="alertdialog" aria-label={message}>
      <span className={styles.message}>{message}</span>
      <div className={styles.actions}>
        <button className={styles.confirmButton} onClick={onConfirm} type="button">
          {intl.formatMessage(messages.confirm)}
        </button>
        <button className={styles.cancelButton} onClick={onCancel} type="button">
          {intl.formatMessage(messages.cancel)}
        </button>
      </div>
    </div>
  );
}
