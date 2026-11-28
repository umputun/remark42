import { h } from 'preact';
import { useLayoutEffect, useRef } from 'preact/hooks';

import { useIntl } from 'common/intl';
import type { StoreState } from 'store';
import { useAppDispatch, useAppSelector } from 'store';
import { updateSorting } from 'store/comments/actions';

import { mountSortPicker } from './sort-picker-element.js';

/**
 * Mounts the plain sort picker into a span preact renders without props or children, so a
 * re-render of the page leaves the span's class and content to the plain component.
 */
export function SortPicker() {
  const intl = useIntl();
  const dispatch = useAppDispatch();
  const sort = useAppSelector((s: StoreState) => s.comments.sort);
  const root = useRef<HTMLSpanElement>(null);
  const picker = useRef<ReturnType<typeof mountSortPicker> | null>(null);
  const props: Parameters<typeof mountSortPicker>[1] = { sort, onChange: (next) => dispatch(updateSorting(next)) };

  useLayoutEffect(() => {
    picker.current = mountSortPicker(root.current!, props, intl);
    // mounted once; the props and intl of every later render reach it through update below
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // every render, so a change of order in the store moves the selection
  useLayoutEffect(() => {
    picker.current?.update(props, intl);
  });

  return <span ref={root} />;
}
