import { h } from 'preact';
import { FormattedMessage, defineMessages, useIntl } from 'react-intl';
import { useMemo } from 'preact/hooks';
import { useSelector, useDispatch } from 'react-redux';
import clsx from 'clsx';

import { StoreState } from 'store';
import { Select } from 'components/select';
import { updateSorting } from 'store/comments/actions';
import type { Sorting } from 'common/types';

import styles from './sort-picker.module.css';

export function SortPicker() {
  const dispatch = useDispatch();
  const sort = useSelector((s: StoreState) => s.comments.sort);
  const intl = useIntl();
  const [items, itemsById] = useMemo(() => {
    const sortOptions = {
      '-score': intl.formatMessage(messages.best),
      '+score': intl.formatMessage(messages.worst),
      '-time': intl.formatMessage(messages.newest),
      '+time': intl.formatMessage(messages.oldest),
      '-active': intl.formatMessage(messages.recentlyUpdated),
      '+active': intl.formatMessage(messages.leastRecentlyUpdated),
      '-controversy': intl.formatMessage(messages.mostControversial),
      '+controversy': intl.formatMessage(messages.leastControversial),
    };
    type SortItem = { value: string; label: string };
    const sort: SortItem[] = Object.entries(sortOptions).map(([k, v]) => ({ value: k, label: v }));
    const sortById = sort.reduce(
      (accum, s) => ({ ...accum, [s.value]: s }),
      {} as Record<keyof typeof sortOptions, SortItem>
    );

    return [sort, sortById];
  }, []);

  function handleSortChange(evt: Event) {
    const { value } = evt.target as HTMLOptionElement;

    if (!(value in itemsById)) {
      return;
    }

    dispatch(updateSorting(value as Sorting));
  }

  return (
    <span className={clsx('sort-picker', styles.root)}>
      <FormattedMessage id="sort-by" defaultMessage="Sort by" />{' '}
      <Select items={items} selected={itemsById[sort]} onChange={handleSortChange} />
    </span>
  );
}

const messages = defineMessages({
  best: {
    id: 'commentsSort.best',
    defaultMessage: 'Best',
  },
  worst: {
    id: 'commentsSort.worst',
    defaultMessage: 'Worst',
  },
  newest: {
    id: 'commentsSort.newest',
    defaultMessage: 'Newest',
  },
  oldest: {
    id: 'commentsSort.oldest',
    defaultMessage: 'Oldest',
  },
  recentlyUpdated: {
    id: 'commentsSort.recently-updated',
    defaultMessage: 'Recently updated',
  },
  leastRecentlyUpdated: {
    id: 'commentsSort.least-recently-updated',
    defaultMessage: 'Least recently updated',
  },
  mostControversial: {
    id: 'commentsSort.most-controversial',
    defaultMessage: 'Most controversial',
  },
  leastControversial: {
    id: 'commentsSort.least-controversial',
    defaultMessage: 'Least controversial',
  },
});
