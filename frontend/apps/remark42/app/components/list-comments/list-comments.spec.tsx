import '@testing-library/jest-dom';

import { render } from 'tests/utils';

import { ListComments } from './list-comments';

describe('<ListComments />', () => {
  it('sets dir on the root element from the active locale (direction mapping itself is covered by common/direction.test.ts)', () => {
    const { container } = render(<ListComments comments={[]} />, {}, 'he');

    expect(container.firstElementChild).toHaveAttribute('dir', 'rtl');
  });
});
