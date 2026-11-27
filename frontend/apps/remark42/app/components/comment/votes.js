// The vote buttons and score of a comment, in plain JavaScript. mountVotes fills an element the
// caller owns and keeps it current: update() changes the nodes in place, so a button keeps its
// hover state and transition across a vote, and destroy() stops the timers and any late answer.

import { defineMessages } from '../../common/intl-message.ts';
import { classes, placeChildren } from '../../common/dom.js';
import { extractErrorMessageFromResponse } from '../../utils/errorUtils.ts';
import { arrowIcon } from '../icons/arrow-icon.js';
import tooltipStyles from '../tooltip/tooltip.module.css';
import styles from './comment-votes.module.css';

// how long the new score shows as pending after the server confirmed it, and how long an error
// stays up when the pointer does not leave it first
const SETTLE_MS = 200;
const ERROR_MS = 10000;

/**
 * @typedef {object} VotesProps
 * @property {string} id
 * @property {0 | -1 | 1} vote  the reader's own vote
 * @property {number} votes  the score
 * @property {number | undefined} controversy
 * @property {boolean} [disabled]
 */

/**
 * @typedef {object} VotesDeps
 * @property {import('../../common/intl-message.ts').IntlShape} intl
 * @property {(id: string, vote: -1 | 1) => Promise<{ id: string, score: number }>} putVote
 * @property {(patch: { id: string, score: number, vote: 0 | -1 | 1 }) => void} patch  stores the result
 * @property {() => boolean} positiveScore  whether a score may not go below zero
 */

/**
 * @param {HTMLElement} root  an empty element; its class and children are this component's
 * @param {VotesProps} initial
 * @param {VotesDeps} initialDeps
 */
export function mountVotes(root, initial, initialDeps) {
  let props = initial;
  let deps = initialDeps;
  /** @type {{ vote: number, votes: number } | null} */
  let loading = null;
  /** @type {string | undefined} */
  let error;
  let destroyed = false;
  /** @type {ReturnType<typeof setTimeout> | undefined} */
  let settleTimer;
  /** @type {ReturnType<typeof setTimeout> | undefined} */
  let errorTimer;

  const down = voteButton(-1, styles.downVoteIcon);
  const up = voteButton(1, styles.upVoteIcon);
  const score = document.createElement('div');
  const tooltipRoot = document.createElement('div');
  tooltipRoot.className = tooltipStyles.root;
  tooltipRoot.append(score);

  const tooltip = document.createElement('div');
  tooltip.setAttribute('role', 'tooltip');
  tooltip.className = classes(tooltipStyles.tooltip, tooltipStyles.tooltipPermanent, tooltipStyles['top-left']);
  const errorText = document.createElement('div');
  errorText.className = styles.errorMessage;
  tooltip.append(errorText);
  tooltip.addEventListener('mouseleave', hideError);

  down.addEventListener('click', () => vote(-1));
  up.addEventListener('click', () => vote(1));

  render();

  /** @param {-1 | 1} increment */
  async function vote(increment) {
    const { id, vote: was, votes } = props;

    loading = { vote: was + increment, votes: votes + increment };
    render();
    try {
      const result = await deps.putVote(id, increment);
      deps.patch({ ...result, vote: /** @type {0 | -1 | 1} */ (was + increment) });
      if (destroyed) {
        return;
      }
      error = undefined;
      clearTimeout(settleTimer);
      settleTimer = setTimeout(() => {
        loading = null;
        render();
      }, SETTLE_MS);
    } catch (err) {
      if (destroyed) {
        return;
      }
      error = extractErrorMessageFromResponse(/** @type {any} */ (err), deps.intl);
      loading = null;
      clearTimeout(errorTimer);
      errorTimer = setTimeout(hideError, ERROR_MS);
    }
    render();
  }

  function hideError() {
    error = undefined;
    render();
  }

  function render() {
    if (destroyed) {
      return;
    }
    const { intl } = deps;
    const { vote: own, votes, controversy, disabled } = props;
    const value = loading?.votes ?? votes;
    const positiveScore = deps.positiveScore() && value > -1;

    root.className = classes(styles.root, disabled && styles.rootDisabled);

    down.className = classes(styles.voteButton, styles.downVoteButton, own === -1 && styles.downVoteButtonActive);
    down.title = intl.formatMessage(messages.downvote);
    down.disabled = loading !== null || own === -1 || positiveScore;
    up.className = classes(styles.voteButton, styles.upVoteButton, own === 1 && styles.upVoteButtonActive);
    up.title = intl.formatMessage(messages.upvote);
    up.disabled = loading !== null || own === 1;

    score.title = controversy
      ? intl.formatMessage(messages.controversy, { value: controversy.toFixed(2) })
      : intl.formatMessage(messages.score);
    score.className = classes(styles.votes, votes < 0 && styles.votesNegative, votes > 0 && styles.votesPositive);
    score.textContent = String(value);

    if (error) {
      errorText.textContent = error;
    }
    placeChildren(tooltipRoot, error ? [tooltip, score] : [score]);
    placeChildren(root, disabled ? [tooltipRoot] : [down, tooltipRoot, up]);
  }

  return {
    /**
     * @param {VotesProps} next
     * @param {VotesDeps} [nextDeps]
     */
    update(next, nextDeps = deps) {
      props = next;
      deps = nextDeps;
      render();
    },
    destroy() {
      destroyed = true;
      clearTimeout(settleTimer);
      clearTimeout(errorTimer);
    },
  };
}

/**
 * @param {-1 | 1} value
 * @param {string} iconClass
 */
function voteButton(value, iconClass) {
  const button = document.createElement('button');
  // class, value and title first, in the order the preact component wrote them
  button.className = '';
  button.dataset.value = String(value);
  button.title = '';
  button.append(arrowIcon(14, { class: iconClass }));
  return button;
}

export const messages = defineMessages({
  score: {
    id: 'vote.score',
    defaultMessage: 'Score',
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
