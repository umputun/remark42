import '@testing-library/jest-dom';
import { h } from 'preact';
import { render } from 'tests/utils';
import { screen } from '@testing-library/preact';

import { Avatar } from './avatar';
import { BASE_URL } from 'common/constants.config';

describe('<Avatar/>', () => {
  it('should have static class name', () => {
    render(<Avatar title="User Name" />);

    expect(screen.getByTitle('User Name')).toHaveClass('avatar');
  });

  it('should have correct url', () => {
    render(<Avatar title="User Name" />);

    expect(screen.getByTitle('User Name')).toHaveAttribute('src', `${BASE_URL}/image.svg`);
  });

  it('should not be accessible with screen reader', () => {
    render(<Avatar title="User Name" />);

    expect(screen.getByTitle('User Name')).toHaveAttribute('aria-hidden', 'true');
  });
});
