import '@testing-library/jest-dom';
import { h } from 'preact';
import { render } from '@testing-library/preact';

import { BASE_URL } from 'common/constants.config';

import { Avatar } from './avatar';

describe('<Avatar/>', () => {
  it('should get', () => {
    const { getByAltText } = render(<Avatar className="avatar" alt="avatar" />);

    expect(getByAltText('avatar')).toHaveAttribute('src', `${BASE_URL}/image.svg`);
    expect(getByAltText('avatar')).toHaveAttribute('alt', 'avatar');
  });
});
