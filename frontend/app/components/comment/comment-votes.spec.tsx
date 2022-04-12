import { h } from 'preact';
import '@testing-library/jest-dom';
import { fireEvent, screen, waitFor } from '@testing-library/preact';
import { render } from 'tests/utils';

import * as api from 'common/api';
import { CommentVotes } from './comment-votes';
import { StaticStore } from 'common/static-store';

describe('<CommentVote />', () => {
  it('should render vote component', () => {
    render(<CommentVotes id="1" vote={0} votes={0} controversy={0} />);
    expect(screen.getByTitle('Vote up')).toBeVisible();
    expect(screen.getByTitle('Vote down')).toBeVisible();
    expect(screen.getByTitle('Votes score')).toBeVisible();
    // expect(screen.getByTitle('Votes score')).toHaveAttribute('title', '0.00');
  });

  it('should render vote component with positive score', () => {
    render(<CommentVotes id="1" vote={0} votes={1} controversy={0} />);
    expect(screen.getByTitle('Votes score')).toBeVisible();
  });
  it('should render vote component with negative score', () => {
    render(<CommentVotes id="1" vote={0} votes={-1} controversy={0} />);
    expect(screen.getByTitle('Votes score')).toBeVisible();
  });

  it('should disable buttons after upvote when request is in progress', () => {
    jest.spyOn(api, 'putCommentVote').mockImplementationOnce(jest.fn(() => new Promise(() => {})));
    render(<CommentVotes id="1" vote={0} votes={10} controversy={0} />);
    fireEvent(screen.getByTitle('Vote up'), new Event('click'));
    expect(screen.getByTitle('Vote down')).toBeDisabled();
    expect(screen.getByTitle('Vote up')).toBeDisabled();
  });

  it('should disable upvote button when upvoted', () => {
    render(<CommentVotes id="1" vote={1} votes={10} controversy={0} />);
    expect(screen.getByTitle('Vote up')).toBeDisabled();
  });

  it('should disable downvote button when downvoted', () => {
    render(<CommentVotes id="1" vote={-1} votes={10} controversy={0} />);
    expect(screen.getByTitle('Vote down')).toBeDisabled();
  });

  it('should disable buttons after downvote when request is in progress', async () => {
    jest.spyOn(api, 'putCommentVote').mockImplementationOnce(jest.fn(() => new Promise(() => {})));
    render(<CommentVotes id="1" vote={0} votes={10} controversy={0} />);
    fireEvent(screen.getByTitle('Vote down'), new Event('click'));
    expect(screen.getByTitle('Vote down')).toBeDisabled();
    expect(screen.getByTitle('Vote up')).toBeDisabled();
  });

  it.each([
    ['upvote', 1, 'Vote up', 'Vote down', 'upVoteButtonActive'],
    ['downvote', -1, 'Vote down', 'Vote up', 'downVoteButtonActive'],
  ])(
    'should go throught voting process and communicate with store when %s button is clicked',
    async (_, increment, activeButtonText, secondButtonText, activeButtonClass) => {
      const putCommentVoteSpy = jest
        .spyOn(api, 'putCommentVote')
        .mockImplementationOnce(({ vote }) => Promise.resolve({ id: '1', score: 10 + vote }));
      render(<CommentVotes id="1" vote={0} votes={10} controversy={0} />);
      fireEvent(screen.getByTitle(activeButtonText), new Event('click'));
      expect(screen.getByTitle('Votes score')).toHaveTextContent(`${10 + increment}`);
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
    render(<CommentVotes id="1" vote={0} votes={10} controversy={0} />);
    expect(screen.getByTitle('Votes score')).toHaveTextContent('10');
    fireEvent(screen.getByTitle('Vote down'), new Event('click'));
    await waitFor(() => expect(screen.getByTitle('Votes score')).toHaveTextContent('9'));
    reject({ code: 0 });
    await waitFor(() => {
      expect(screen.getByTitle('Votes score')).toHaveTextContent('10');
      expect(screen.getByText('Something went wrong. Please try again a bit later.')).toBeVisible();
    });
    putCommnetVoteSpy.mockClear();
  });

  it('should render without voting buttons', () => {
    render(<CommentVotes id="1" vote={0} votes={10} controversy={0} disabled={true} />);
    expect(screen.queryByTitle('Vote down')).not.toBeInTheDocument();
    expect(screen.queryByTitle('Vote up')).not.toBeInTheDocument();
  });

  it('should allow only upvote ability', () => {
    StaticStore.config.positive_score = true;
    render(<CommentVotes id="1" vote={0} votes={10} controversy={0} />);
    expect(screen.queryByTitle('Vote down')).not.toBeInTheDocument();
    expect(screen.getByTitle('Vote up')).toBeVisible();
  });

  it('should disable downvote ability when `low_score` is reached', async () => {
    StaticStore.config.low_score = -4;
    jest.spyOn(api, 'putCommentVote').mockImplementation(jest.fn(async () => ({ id: '1', score: -4 })));
    render(<CommentVotes id="1" vote={0} votes={-3} controversy={0} />);
    expect(screen.getByTitle('Vote down')).not.toBeDisabled();
    fireEvent(screen.getByTitle('Vote down'), new Event('click'));
    await waitFor(() => {
      expect(screen.getByTitle('Vote down')).toBeDisabled();
    });
  });
});
