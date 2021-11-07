import '@testing-library/jest-dom';
import { fireEvent } from '@testing-library/preact';
import { h } from 'preact';
import { render } from 'tests/utils';
import { Select } from './select';

const items = [
  { label: 'None', value: 'none' },
  { label: 'Oldest', value: 'oldest' },
  { label: 'Newest', value: 'newest' },
  { label: 'Best', value: 'best' },
  { label: 'Worst', value: 'worst' },
];

describe('<Select/>', () => {
  it('should has static class names', () => {
    const { container } = render(<Select items={items} selected={items[0]} />);
    const selectElement = container.querySelector('.select-element');

    expect(container.querySelector('.select')).toBeInTheDocument();
    expect(container.querySelector('.select-arrow')).toBeInTheDocument();
    expect(selectElement).toBeInTheDocument();
    fireEvent.focus(selectElement as HTMLSelectElement);
    expect(container.querySelector('.select_focused')).toBeInTheDocument();
  });

  it('should render selected item', () => {
    const { container, getAllByText } = render(<Select items={items} selected={items[0]} />);
    const selectedOption = container.querySelector('option');

    expect(getAllByText(items[0].label)).toHaveLength(2);
    expect(selectedOption).toBeInTheDocument();
    expect(selectedOption?.selected).toBeTruthy();
    expect(selectedOption?.textContent).toBe(items[0].label);
  });

  it('should highlight select on focus', async () => {
    const { container } = render(<Select items={items} selected={items[0]} />);
    const select = container.querySelector('select');

    expect(container.querySelector('select')).toBeInTheDocument();
    fireEvent.focus(select as HTMLSelectElement);
    expect(container.querySelector('.rootFocused')).toBeInTheDocument();
  });
});
