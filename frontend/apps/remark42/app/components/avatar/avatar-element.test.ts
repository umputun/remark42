import '@testing-library/jest-dom';

import { avatarElement } from './avatar-element.js';

/** the avatar in a container, as a page holds it */
function render(props: Parameters<typeof avatarElement>[0]) {
  const container = document.createElement('div');
  container.append(avatarElement(props).img);
  return { container };
}

describe('the avatar', () => {
  it('should have correct url', () => {
    const { container } = render({ size: 24 });

    // the server embeds ghost.svg and serves it under /web of the configured host
    expect(container.querySelector('img')).toHaveAttribute('src', 'http://test.com/web/ghost.svg');
  });

  it('should render an identicon pixelated', () => {
    const { container } = render({ url: 'http://localhost/avatar.image', size: 24, identicon: true });

    expect(container.querySelector('img')).toHaveClass('avatarIdenticon');
  });

  it.each([
    ['a picture that is not an identicon', { url: 'http://localhost/avatar.image' }],
    ['the placeholder', { identicon: true }],
  ])('should render %s smoothly', (_, props) => {
    const { container } = render({ size: 24, ...props });

    expect(container.querySelector('img')).not.toHaveClass('avatarIdenticon');
  });

  it('should carry its size and an empty alt', () => {
    const { container } = render({ url: 'http://localhost/avatar.image', size: 24 });
    const img = container.querySelector('img');

    expect(img).toHaveAttribute('width', '24');
    expect(img).toHaveAttribute('height', '24');
    expect(img).toHaveAttribute('alt', '');
    expect(img).toHaveAttribute('decoding', 'async');
  });

  it("shouldn't be accessible with screen reader", () => {
    const { container } = render({ size: 24 });

    expect(container.querySelector('img')).toHaveAttribute('aria-hidden', 'true');
  });
});
