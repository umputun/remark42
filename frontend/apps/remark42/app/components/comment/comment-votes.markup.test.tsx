import { h } from 'preact';
import { act, fireEvent } from '@testing-library/preact';

import * as api from 'common/api';
import { StaticStore } from 'common/static-store';
import { render } from 'tests/utils';

import { CommentVotes } from './comment-votes';

type Props = Parameters<typeof CommentVotes>[0];

const base: Props = { id: 'c1', vote: 0, votes: 3, controversy: undefined };

afterEach(() => {
  StaticStore.config.positive_score = false;
  jest.restoreAllMocks();
});

// the markup the preact component drew, recorded before it was replaced; innerHTML keeps the
// attributes in the order they were written
describe('the vote buttons draw the markup of the preact component they replace', () => {
  it.each<[string, Partial<Props>, (() => void)?]>([
    ['no vote, positive score', {}],
    ['upvoted', { vote: 1, votes: 4 }],
    ['downvoted, negative score', { vote: -1, votes: -2 }],
    ['zero score', { votes: 0 }],
    ['with controversy', { controversy: 1.2345 }],
    ['disabled', { disabled: true }],
    [
      'positive score only, at zero',
      { votes: 0 },
      () => {
        StaticStore.config.positive_score = true;
      },
    ],
  ])('%s', (_, change, setup) => {
    setup?.();
    expect(render(<CommentVotes {...base} {...change} />).container.innerHTML).toMatchSnapshot();
  });

  it('while a vote is in flight, and once it failed', async () => {
    let fail = (_: unknown) => {};
    jest.spyOn(api, 'putCommentVote').mockImplementation(() => new Promise((_, reject) => (fail = reject)));
    // from zero, so the pending score differs in sign from the one the colour follows
    const { container } = render(<CommentVotes {...base} votes={0} />);

    fireEvent.click(container.querySelector('[title="Vote up"]')!);
    expect(container.innerHTML).toMatchSnapshot('in flight');

    await act(async () => fail({ code: 0 }));
    expect(container.innerHTML).toMatchSnapshot('failed');
  });
});
