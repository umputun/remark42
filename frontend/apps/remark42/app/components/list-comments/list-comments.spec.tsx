import '@testing-library/jest-dom';

import { render } from 'tests/utils';

import type { Comment } from 'common/types';

import { ListComments } from './list-comments';

describe('<ListComments />', () => {
  it('sets dir on the root element from the active locale (direction mapping itself is covered by common/direction.test.ts)', () => {
    const { container } = render(<ListComments comments={[]} />, {}, 'he');

    expect(container.firstElementChild).toHaveAttribute('dir', 'rtl');
  });

  it("keeps the commenter's name, and not the comment text, out of search snippets", () => {
    const comment: Comment = {
      id: 'c1',
      pid: '',
      text: '<p>the comment text</p>',
      orig: 'the comment text',
      score: 0,
      vote: 0,
      voted_ips: [],
      time: '2026-11-21T18:05:00Z',
      locator: { url: 'http://localhost/post/1', site: 'remark' },
      user: { id: 'user', name: 'commenter', picture: '', ip: '', admin: false, block: false, verified: false },
    };

    const { container, getByText } = render(<ListComments comments={[comment]} />);

    expect(getByText('commenter').closest('[data-nosnippet]')?.tagName).toBe('DIV');
    expect(container.querySelector('.raw-content')).not.toBeNull();
    expect(container.querySelector('.raw-content')?.closest('[data-nosnippet]')).toBeNull();
  });
});
