import { h } from 'preact';
import '@testing-library/jest-dom';
import { screen } from '@testing-library/preact';
import { render } from 'tests/utils';
import { ArrowIcon } from './arrow';

describe('<ArrowIcon />', () => {
  it('should be rendered with default size', async () => {
    render(<ArrowIcon title="icon" />);
    expect(await screen.findByTitle('icon')).toHaveAttribute('width', '14');
    expect(await screen.findByTitle('icon')).toHaveAttribute('height', '14');
  });
  it('should be rendered with provided size', async () => {
    render(<ArrowIcon title="icon" size={16} />);
    expect(await screen.findByTitle('icon')).toHaveAttribute('width', '16');
    expect(await screen.findByTitle('icon')).toHaveAttribute('height', '16');
  });
});
