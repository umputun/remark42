import '@testing-library/jest-dom';
import { h } from 'preact';
import { render } from 'tests/utils';
import { Select } from './select';

const items = [
  {
    label: 'None',
    value: 'none',
  },
  {
    label: 'Oldest',
    value: 'oldest',
  },
  {
    label: 'Newest',
    value: 'newest',
  },
  {
    label: 'Best',
    value: 'best',
  },
  {
    label: 'Worst',
    value: 'worst',
  },
];

describe('<Select/>', () => {
  it('should has static class names', () => {
    const { container } = render(<Select items={items} selected={items[0]} />);

    expect(container.querySelector('.select')).toBeInTheDocument();
    expect(container.querySelector('.select-arrow')).toBeInTheDocument();
    expect(container.querySelector('.select-element')).toBeInTheDocument();
  });

  it('should render selected item', () => {
    const { container, getAllByText } = render(<Select items={items} selected={items[0]} />);
    const selectedOption = container.querySelectorAll('option')[0];

    expect(getAllByText(items[0].label)).toHaveLength(2);
    expect(selectedOption.selected).toBeTruthy();
    expect(selectedOption.textContent).toBe(items[0].label);
  });
});
