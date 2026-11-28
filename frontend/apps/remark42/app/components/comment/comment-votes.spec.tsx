import '@testing-library/jest-dom';
import { fireEvent, screen, waitFor } from '@testing-library/preact';

import * as api from 'common/api';
import { createIntl } from 'common/intl';
import en from 'locales/en.json';
import { StaticStore } from 'common/static-store';
import { draw } from 'tests/plain';

import { mountVotes } from './votes.js';

function drawVotes(props: Parameters<typeof mountVotes>[1]) {
  return draw('span', (root) =>
    mountVotes(root, props, {
      intl: createIntl('en', en),
      putVote: (id, vote) => api.putCommentVote({ id, vote }),
      patch: () => {},
      positiveScore: () => Boolean(StaticStore.config.positive_score),
    })
  );
}

describe('<CommentVote />', () => {
  it('should disable downvote button when downvoted', () => {
    drawVotes({ id: '1', vote: -1, votes: 10, controversy: 0 });
    expect(screen.getByTitle('Vote down')).toBeDisabled();
  });

  it.each([
    ['upvote', 1, 'Vote up', 'Vote down', 'upVoteButtonActive'],
    ['downvote', -1, 'Vote down', 'Vote up', 'downVoteButtonActive'],
  ])(
    'should go throught voting process and communicate with store when %s button is clicked',
    async (_, increment, activeButtonText, secondButtonText) => {
      const putCommentVoteSpy = jest
        .spyOn(api, 'putCommentVote')
        .mockImplementationOnce(({ vote }) => Promise.resolve({ id: '1', score: 10 + vote }));
      drawVotes({ id: '1', vote: 0, votes: 10, controversy: 0 });
      fireEvent(screen.getByTitle(activeButtonText), new Event('click'));
      expect(screen.getByTitle('Score')).toHaveTextContent(`${10 + increment}`);
      expect(screen.getByTitle(activeButtonText)).toBeDisabled();
      expect(screen.getByTitle(secondButtonText)).toBeDisabled();
      await waitFor(() => expect(putCommentVoteSpy).toHaveBeenCalledWith({ id: '1', vote: increment }));
    }
  );
  it('should render tooltip with error when request failed', async () => {
    let reject = (_: { code: number }) => {};
    const putCommnetVoteSpy = jest.spyOn(api, 'putCommentVote').mockImplementationOnce(
      jest.fn(
        () =>
          new Promise((_, r) => {
            reject = r;
          }) as Promise<{ id: string; score: number }>
      )
    );
    drawVotes({ id: '1', vote: 0, votes: 10, controversy: 0 });
    expect(screen.getByTitle('Score')).toHaveTextContent('10');
    fireEvent(screen.getByTitle('Vote down'), new Event('click'));
    await waitFor(() => expect(screen.getByTitle('Score')).toHaveTextContent('9'));
    reject({ code: 0 });
    await waitFor(() => {
      expect(screen.getByTitle('Score')).toHaveTextContent('10');
      expect(screen.getByText('Something went wrong. Please try again a bit later.')).toBeVisible();
    });
    putCommnetVoteSpy.mockClear();
  });

  it('should render without voting buttons', () => {
    drawVotes({ id: '1', vote: 0, votes: 10, controversy: 0, disabled: true });
    expect(screen.queryByTitle('Vote down')).not.toBeInTheDocument();
    expect(screen.queryByTitle('Vote up')).not.toBeInTheDocument();
  });

  it('should allow only upvote ability', () => {
    StaticStore.config.positive_score = true;
    drawVotes({ id: '1', vote: 0, votes: 10, controversy: 0 });
    expect(screen.queryByTitle('Vote down')).toBeVisible();
    expect(screen.queryByTitle('Vote down')).toBeDisabled();
    expect(screen.getByTitle('Vote up')).toBeVisible();
    expect(screen.getByTitle('Vote up')).not.toBeDisabled();
  });

  it('shows the vote count as the tooltip when there is no controversy', () => {
    drawVotes({ id: '1', vote: 0, votes: 3, controversy: 0 });

    expect(screen.getByTitle('Score')).toHaveTextContent('3');
    expect(screen.queryByTitle(/Controversy/)).not.toBeInTheDocument();
  });

  // what actually arrives: the backend marks the field omitempty, so an uncontroversial
  // comment carries no controversy at all rather than a zero
  it('shows the vote count as the tooltip when controversy is absent', () => {
    drawVotes({ id: '1', vote: 0, votes: 3, controversy: undefined });

    expect(screen.getByTitle('Score')).toHaveTextContent('3');
    expect(screen.queryByTitle(/Controversy/)).not.toBeInTheDocument();
  });

  it('shows the controversy as the tooltip when there is some', () => {
    drawVotes({ id: '1', vote: 0, votes: 3, controversy: 1.234 });

    expect(screen.getByTitle('Controversy: 1.23')).toHaveTextContent('3');
    expect(screen.queryByTitle('Score')).not.toBeInTheDocument();
  });
});
