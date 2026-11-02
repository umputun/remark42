import '@testing-library/jest-dom';
import { render } from '@testing-library/preact';

import { Avatar } from './avatar';

describe('<Avatar/>', () => {
  it('should have correct url', () => {
    const { container } = render(<Avatar />);

    // the asset url already carries the public path; prefixing BASE_URL onto it doubles the origin
    expect(container.querySelector('img')).toHaveAttribute('src', 'http://localhost:8080/web/image.svg');
  });

  it("shouldn't be accessible with screen reader", () => {
    const { container } = render(<Avatar />);

    expect(container.querySelector('img')).toHaveAttribute('aria-hidden', 'true');
  });
});
