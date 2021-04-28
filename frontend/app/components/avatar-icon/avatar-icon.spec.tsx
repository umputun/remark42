import '@testing-library/jest-dom';
import { h } from 'preact';
import { render } from '@testing-library/preact';
import { AvatarIcon } from './avatar-icon';

describe('<AvatarIcon/>', () => {
  it('should get', () => {
    const { getByAltText } = render(<AvatarIcon alt="avatar" />);

    expect(getByAltText('avatar')).toHaveAttribute('src', 'https://url-to-svg-image');
  });
});
