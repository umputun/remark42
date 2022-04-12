import { h } from 'preact';
import '@testing-library/jest-dom';
import { render } from 'tests/utils';

import { Tooltip } from './tooltip';
import { screen } from '@testing-library/preact';

describe('<Tooltip />', () => {
  it('should not render tooltip without content', () => {
    render(<Tooltip position="top-left">Hello</Tooltip>);
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument();
  });

  it('should not render tooltip with content', () => {
    render(
      <Tooltip position="top-left" content="Howdy">
        Hello
      </Tooltip>
    );
    expect(screen.queryByRole('tooltip')).toBeInTheDocument();
    expect(screen.getByText('Howdy')).toBeInTheDocument();
  });

  it.each([['top-left'], ['top-right']] as ['top-left' | 'top-right'][])('should render tooltip on %s', (position) => {
    render(
      <Tooltip position={position} content="Howdy">
        Hello
      </Tooltip>
    );
    expect(screen.queryByRole('tooltip')).toHaveClass(position);
  });
});
