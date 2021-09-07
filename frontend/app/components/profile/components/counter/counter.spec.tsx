import { h } from 'preact';
import '@testing-library/jest-dom';
import { render } from 'tests/utils';
import { Counter } from '.';

describe('Counter', () => {
  it('renders correctly', () => {
    const children = 11;
    const { getByText } = render(<Counter>{children}</Counter>);

    expect(getByText(children)).toBeInTheDocument();
  });
});
