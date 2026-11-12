import { h } from 'preact';
import { useEffect, useId, useRef } from 'preact/hooks';
import { useIntl, defineMessages } from 'common/intl';

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
  const messageId = useId();
  const cancelRef = useRef<HTMLButtonElement>(null);

  // move focus into the dialog so keyboard and screen reader users land on it; cancel is the safe default
  useEffect(() => {
    cancelRef.current?.focus();
  }, []);

  function handleKeyDown(evt: KeyboardEvent) {
    if (evt.key === 'Escape') {
      evt.stopPropagation();
      onCancel();
    }
  }

  return (
    <div className={styles.root} role="alertdialog" aria-labelledby={messageId}>
      <span id={messageId} className={styles.message}>
        {message}
      </span>
      <div className={styles.actions}>
        <button className={styles.confirmButton} onClick={onConfirm} onKeyDown={handleKeyDown} type="button">
          {intl.formatMessage(messages.confirm)}
        </button>
        <button
          ref={cancelRef}
          className={styles.cancelButton}
          onClick={onCancel}
          onKeyDown={handleKeyDown}
          type="button"
        >
          {intl.formatMessage(messages.cancel)}
        </button>
      </div>
    </div>
  );
}
