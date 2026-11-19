import '@testing-library/jest-dom';
import { render } from '@testing-library/preact';

import { Avatar } from './avatar';

describe('<Avatar/>', () => {
  it('should have correct url', () => {
    const { container } = render(<Avatar />);

    // the server embeds ghost.svg and serves it under /web of the configured host
    expect(container.querySelector('img')).toHaveAttribute('src', 'http://test.com/web/ghost.svg');
  });

  it('should render an identicon pixelated', () => {
    const { container } = render(<Avatar url="http://localhost/avatar.image" identicon />);

    expect(container.querySelector('img')).toHaveClass('avatarIdenticon');
  });

  it.each([
    ['a picture that is not an identicon', { url: 'http://localhost/avatar.image' }],
    ['the placeholder', { identicon: true }],
  ])('should render %s smoothly', (_, props) => {
    const { container } = render(<Avatar {...props} />);

    expect(container.querySelector('img')).not.toHaveClass('avatarIdenticon');
  });

  it("shouldn't be accessible with screen reader", () => {
    const { container } = render(<Avatar />);

    expect(container.querySelector('img')).toHaveAttribute('aria-hidden', 'true');
  });
});
