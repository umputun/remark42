import { h } from 'preact';
import { act, fireEvent } from '@testing-library/preact';

import { render } from 'tests/utils';
import type { Sorting } from 'common/types';
import type { StoreState } from 'store';

import { SortPicker } from './sort-picker';

const sorts: Sorting[] = ['-score', '+score', '-time', '+time', '-active', '+active', '-controversy', '+controversy'];

function withSort(sort: Sorting) {
  return { comments: { sort } } as unknown as Partial<StoreState>;
}

// the markup the preact component drew, recorded before it was replaced; innerHTML keeps the
// attributes in the order they were written
describe('the sort picker draws the markup of the preact component it replaces', () => {
  it.each(sorts)('sorted by %s', (sort) => {
    expect(render(<SortPicker />, withSort(sort)).container.innerHTML).toMatchSnapshot();
  });

  it('focused', () => {
    const { container } = render(<SortPicker />, withSort('-time'));

    act(() => container.querySelector('select')!.focus());
    expect(container.innerHTML).toMatchSnapshot();
  });

  it('after a pick, until the store answers', () => {
    const { container } = render(<SortPicker />, withSort('-time'));
    const select = container.querySelector('select')!;

    select.value = '+score';
    act(() => {
      fireEvent.change(select);
    });
    expect(container.innerHTML).toMatchSnapshot();
  });
});
