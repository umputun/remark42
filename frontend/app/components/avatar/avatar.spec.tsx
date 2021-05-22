import '@testing-library/jest-dom';
import { h } from 'preact';
import { render } from '@testing-library/preact';

import { Avatar } from './avatar';
import { BASE_URL } from 'common/constants.config';

describe('<Avatar/>', () => {
  it('should have correct url', () => {
    const { container } = render(<Avatar />);

    expect(container.querySelector('img')).toHaveAttribute('src', `${BASE_URL}/image.svg`);
  });

  it("shouldn't be accessible with screen reader", () => {
    const { container } = render(<Avatar />);

    expect(container.querySelector('img')).toHaveAttribute('aria-hidden', 'true');
  });
});
