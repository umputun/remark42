import en from 'locales/en.json';
import { createIntl } from 'common/intl-message';

import { mountSortPicker } from './sort-picker-element.js';

const intl = createIntl('en', en);

// the same node objects, one by one: toEqual would also accept equal copies
function expectSameNodes(after: Element[], before: Element[]) {
  expect(after).toHaveLength(before.length);
  after.forEach((node, i) => expect(node).toBe(before[i]));
}

function mount(sort: Parameters<typeof mountSortPicker>[1]['sort'] = '-time') {
  const root = document.createElement('span');
  const onChange = jest.fn();
  const picker = mountSortPicker(root, { sort, onChange }, intl);
  const select = root.querySelector('select') as HTMLSelectElement;
  const pick = (value: string) => {
    select.value = value;
    select.dispatchEvent(new Event('change'));
  };
  return { root, picker, select, pick, onChange };
}

describe('mountSortPicker', () => {
  it('reports a pick to the handler of the latest props, and ignores an unknown order', () => {
    const { picker, pick, onChange } = mount();
    const latest = jest.fn();
    picker.update({ sort: '-time', onChange: latest });

    pick('+score');
    expect(latest).toHaveBeenCalledWith('+score');
    expect(onChange).not.toHaveBeenCalled();

    const option = document.createElement('option');
    option.value = 'sideways';
    (mount().select as HTMLSelectElement).append(option);
    latest.mockClear();
    const other = mount();
    other.select.append(option);
    other.pick('sideways');
    expect(other.onChange).not.toHaveBeenCalled();
  });

  it('keeps the pick until the store answers, then follows the store', () => {
    const { root, picker, select, pick } = mount('-time');

    pick('+score');
    picker.update({ sort: '-time', onChange: jest.fn() });
    expect(select.value).toBe('+score');
    expect(root.querySelector('[data-testid="select-root"]')!.firstChild!.textContent).toBe('Newest');

    picker.update({ sort: '+score', onChange: jest.fn() });
    expect(select.value).toBe('+score');
    expect(root.querySelector('[data-testid="select-root"]')!.firstChild!.textContent).toBe('Worst');

    picker.update({ sort: '-active', onChange: jest.fn() });
    expect(select.value).toBe('-active');
  });

  it('keeps its nodes across updates', () => {
    const { root, picker } = mount();
    const before = Array.from(root.querySelectorAll('*'));

    picker.update({ sort: '+time', onChange: jest.fn() });

    expectSameNodes(Array.from(root.querySelectorAll('*')), before);
  });
});
