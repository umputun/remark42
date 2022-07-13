import clsx from 'clsx';
import { h, Fragment } from 'preact';
import { defineMessages, useIntl } from 'react-intl';

import { BlockTTL } from 'common/types';
import { Select } from 'components/select';
import { Countdown } from 'components/countdown';
import { Button } from 'components/auth/components/button';

import { getBlockingDurations } from './getBlockingDurations';
import styles from './comment-actions.module.css';

export type Props = {
  admin: boolean | undefined;
  currentUser: boolean | undefined;
  pinned: boolean | undefined;
  copied: boolean | undefined;
  bannedUser: boolean | undefined;
  readOnly: boolean | undefined;
  editing: boolean | undefined;
  replying: boolean | undefined;
  editable: boolean;
  editDeadline: number | undefined;
  onCopy(): void;
  onToggleEditing(): void;
  onDelete(): void;
  onTogglePin(): void;
  onToggleReplying(): void;
  onHideUser(): void;
  onBlockUser(ttl: BlockTTL): void;
  onUnblockUser(): void;
  onDisableEditing(): void;
};

export function CommentActions({
  admin,
  pinned,
  copied,
  readOnly,
  editable,
  editing,
  replying,
  currentUser,
  bannedUser,
  editDeadline,
  onCopy,
  onToggleEditing,
  onDelete,
  onTogglePin,
  onToggleReplying,
  onDisableEditing,
  onHideUser,
  onBlockUser,
  onUnblockUser,
}: Props) {
  const intl = useIntl();

  const deleteJSX = (
    <Button kind="link" size="sm" onClick={onDelete}>
      {intl.formatMessage(messages.delete)}
    </Button>
  );

  return (
    <div className={clsx('comment-actions', styles.root)}>
      {!readOnly && (
        <Button kind="link" size="sm" onClick={onToggleReplying}>
          {intl.formatMessage(replying ? messages.cancel : messages.reply)}
        </Button>
      )}
      {editable && editDeadline && (
        <>
          <Button kind="link" size="sm" onClick={onToggleEditing}>
            {intl.formatMessage(editing ? messages.cancel : messages.edit)}
          </Button>
          <span
            role="timer"
            title={intl.formatMessage(messages.editCountdown)}
            className={clsx('comment-actions-countdown', styles.countdown)}
          >
            <Countdown timestamp={editDeadline} onTimePassed={onDisableEditing} />
          </span>
        </>
      )}
      <div
        data-testid="comment-actions-additional"
        className={clsx('comment-actions-additional', styles.additionalActions)}
      >
        {!currentUser && (
          <Button kind="link" size="sm" onClick={onHideUser}>
            {intl.formatMessage(messages.hide)}
          </Button>
        )}
        {admin && (
          <>
            <Button kind="link" size="sm" onClick={onCopy} disabled={copied}>
              {intl.formatMessage(copied ? messages.copied : messages.copy)}
            </Button>
            <Button kind="link" size="sm" onClick={onTogglePin}>
              {intl.formatMessage(pinned ? messages.unpin : messages.pin)}
            </Button>
            {bannedUser ? (
              <Button kind="link" size="sm" onClick={onUnblockUser}>
                {intl.formatMessage(messages.unblock)}
              </Button>
            ) : (
              <Select
                title={intl.formatMessage(messages.blockingPeriod)}
                size="sm"
                items={getBlockingDurations(intl)}
                onChange={(evt) => onBlockUser(evt.currentTarget.value as BlockTTL)}
              />
            )}
          </>
        )}
        {(currentUser || admin) && deleteJSX}
      </div>
    </div>
  );
}

const messages = defineMessages({
  unblock: { id: 'comment.unblock', defaultMessage: 'Unblock' },
  pin: { id: 'comment.pin', defaultMessage: 'Pin' },
  unpin: { id: 'comment.unpin', defaultMessage: 'Unpin' },
  hide: { id: 'comment.hide', defaultMessage: 'Hide' },
  cancel: { id: 'comment.cancel', defaultMessage: 'Cancel' },
  edit: { id: 'comment.edit', defaultMessage: 'Edit' },
  reply: { id: 'comment.reply', defaultMessage: 'Reply' },
  delete: { id: 'comment.delete', defaultMessage: 'Delete' },
  editCountdown: { id: 'comment.edit-countdown', defaultMessage: 'Edit will be disabled' },
  copied: { id: 'comment.copied', defaultMessage: 'Copied!' },
  copy: { id: 'comment.copy', defaultMessage: 'Copy' },
  blockingPeriod: { id: 'comment.blocking-period', defaultMessage: 'Blocking period' },
});
