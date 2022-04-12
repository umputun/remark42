import clsx from 'clsx';
import { h } from 'preact';
import { useState } from 'preact/hooks';
import { defineMessages, useIntl } from 'react-intl';

import { useDispatch } from 'react-redux';
import { patchComment } from 'store/comments/actions';
import { putCommentVote } from 'common/api';
import { StaticStore } from 'common/static-store';
import { ArrowIcon } from 'components/icons/arrow';

import styles from './comment-votes.module.css';
import { Tooltip } from 'components/tooltip';
import { extractErrorMessageFromResponse } from 'utils/errorUtils';

type Props = {
  id: string;
  vote: 0 | -1 | 1;
  votes: number;
  controversy: number | undefined;
  disabled?: boolean;
};

export function CommentVotes({ id, votes, vote, disabled, controversy = 0 }: Props) {
  const intl = useIntl();
  const dispatch = useDispatch();
  const [loadingState, setLoadingState] = useState<{ vote: number; votes: number } | null>(null);
  const [errorMessage, setErrorMessage] = useState<string | void>(undefined);

  async function handleClick(evt: preact.JSX.TargetedEvent<HTMLButtonElement>) {
    const { value } = evt.currentTarget.dataset;
    const increment = Number(value) as -1 | 1;

    setLoadingState({ vote: vote + increment, votes: votes + increment });
    try {
      const p = await putCommentVote({ id, vote: increment });
      dispatch(patchComment({ ...p, vote: (vote + increment) as -1 | 0 | 1 }));
      setErrorMessage(undefined);
      setTimeout(() => setLoadingState(null), 200);
    } catch (err) {
      // @ts-ignore
      setErrorMessage(extractErrorMessageFromResponse(err, intl));
      setLoadingState(null);
    }
  }

  const lowScore = StaticStore.config.low_score === votes;
  const positiveScore = StaticStore.config.positive_score;
  const isUpvoted = vote === 1;
  const isDownvoted = vote === -1;

  return (
    <span className={clsx(styles.root, disabled && styles.rootDisabled)}>
      {Boolean(!disabled && !positiveScore) && (
        <button
          className={clsx(styles.voteButton, styles.downVoteButton, isDownvoted && styles.downVoteButtonActive)}
          onClick={handleClick}
          data-value={-1}
          title={intl.formatMessage(messages.downvote)}
          disabled={lowScore || loadingState !== null || isDownvoted}
        >
          <ArrowIcon className={styles.downVoteIcon} />
        </button>
      )}
      <Tooltip
        content={errorMessage ? <div class={styles.errorMessage}>{errorMessage}</div> : undefined}
        position="top-left"
        hideBehavior="mouseleave"
        hideTimeout={10000}
        permanent
        onHide={() => {
          setErrorMessage(undefined);
        }}
      >
        <div
          title={intl.formatMessage(messages.score)}
          // title={intl.formatMessage(messages.controversy, { value: controversy })}
          className={clsx(styles.votes, {
            [styles.votesNegative]: votes < 0,
            [styles.votesPositive]: votes > 0,
          })}
        >
          {loadingState?.votes ?? votes}
        </div>
      </Tooltip>
      {!disabled && (
        <button
          className={clsx(styles.voteButton, styles.upVoteButton, isUpvoted && styles.upVoteButtonActive)}
          onClick={handleClick}
          data-value={1}
          title={intl.formatMessage(messages.upvote)}
          disabled={loadingState !== null || isUpvoted}
        >
          <ArrowIcon className={styles.upVoteIcon} />
        </button>
      )}
    </span>
  );
}

export const messages = defineMessages({
  score: {
    id: 'vote.score',
    defaultMessage: 'Votes score',
  },
  upvote: {
    id: 'vote.upvote',
    defaultMessage: 'Vote up',
  },
  downvote: {
    id: 'vote.downvote',
    defaultMessage: 'Vote down',
  },
  controversy: {
    id: 'vote.controversy',
    defaultMessage: 'Controversy: {value}',
  },
});
