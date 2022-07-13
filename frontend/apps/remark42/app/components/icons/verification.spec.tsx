import { h } from 'preact';
import '@testing-library/jest-dom';
import { screen } from '@testing-library/preact';
import { render } from 'tests/utils';
import { VerificationIcon } from './verification';

describe('<VerificationIcon />', () => {
  it('should be rendered with default size', async () => {
    render(<VerificationIcon title="icon" />);
    expect(await screen.findByTitle('icon')).toHaveAttribute('width', '12');
    expect(await screen.findByTitle('icon')).toHaveAttribute('height', '12');
  });
  it('should be rendered with provided size', async () => {
    render(<VerificationIcon title="icon" size={16} />);
    expect(await screen.findByTitle('icon')).toHaveAttribute('width', '16');
    expect(await screen.findByTitle('icon')).toHaveAttribute('height', '16');
  });
});
