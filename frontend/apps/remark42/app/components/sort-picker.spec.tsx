import '@testing-library/jest-dom';
import { screen, fireEvent, waitFor } from '@testing-library/preact';

import { render } from 'tests/utils';
import * as commentsActions from 'store/comments/actions';
import type { StoreState } from 'store';

import { SortPicker } from './sort-picker';

const defaultState = { comments: {} as StoreState['comments'], hiddenUsers: {} };

describe('<SortPicker />', () => {
  it('should render sort picker with options', () => {
    const { container, queryAllByText, queryByText } = render(<SortPicker />, defaultState);

    expect(container.querySelectorAll('option')).toHaveLength(8);
    expect(queryAllByText('Best')).toHaveLength(2);
    expect(queryByText('Sort by')).toBeInTheDocument();
  });

  it('should has static class names', () => {
    const { container } = render(<SortPicker />, defaultState);

    expect(container.querySelector('.sort-picker')).toBeInTheDocument();
  });

  it('should render selected element', () => {
    render(<SortPicker />, { comments: { sort: '-active' } as StoreState['comments'] });
    expect(screen.getAllByText<HTMLOptionElement>('Recently updated')[1].selected).toBeTruthy();
  });

  it('should change selected store', async () => {
    const nextOption = '-controversy';
    const updateSorting = jest.spyOn(commentsActions, 'updateSorting');
    const { container } = render(<SortPicker />, defaultState);
    const select = container.querySelector('select') as HTMLSelectElement;

    expect(select).toBeInTheDocument();

    fireEvent.change(select, { target: { value: nextOption } });

    await waitFor(() => expect(updateSorting).toHaveBeenCalledWith(nextOption));

    expect(container.querySelector<HTMLOptionElement>(`[value="${nextOption}"]`)?.selected).toBeTruthy();
  });
});
