import '@testing-library/jest-dom';
import { h } from 'preact';
import { render } from '@testing-library/preact';

import { BASE_URL } from 'common/constants.config';

import { Avatar } from './avatar';

describe('<Avatar/>', () => {
  it('should have correct url', () => {
    const { container } = render(<Avatar className="avatar" />);

    expect(container.querySelector('img')).toHaveAttribute('src', `${BASE_URL}/image.svg`);
  });

  it("shouldn't be accessible with screen reader", () => {
    const { container } = render(<Avatar className="avatar" />);

    expect(container.querySelector('img')).toHaveAttribute('aria-hidden', 'true');
  });
});
