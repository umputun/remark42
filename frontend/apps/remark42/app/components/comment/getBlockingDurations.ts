import { defineMessages, IntlShape } from 'react-intl';
import { BlockTTL } from 'common/types';

export interface BlockingDuration {
  label: string;
  value: BlockTTL | undefined;
}

const blockingMessages = defineMessages({
  block: {
    id: 'comment.block',
    defaultMessage: 'Block',
  },
  permanently: {
    id: 'blockingDuration.permanently',
    defaultMessage: 'Permanently',
  },
  month: {
    id: 'blockingDuration.month',
    defaultMessage: 'For a month',
  },
  week: {
    id: 'blockingDuration.week',
    defaultMessage: 'For a week',
  },
  day: {
    id: 'blockingDuration.day',
    defaultMessage: 'For a day',
  },
});

export function getBlockingDurations(intl: IntlShape): BlockingDuration[] {
  return [
    {
      label: intl.formatMessage(blockingMessages.block),
      value: undefined,
    },
    {
      label: intl.formatMessage(blockingMessages.permanently),
      value: 'permanently',
    },
    {
      label: intl.formatMessage(blockingMessages.month),
      value: '43200m',
    },
    {
      label: intl.formatMessage(blockingMessages.week),
      value: '10080m',
    },
    {
      label: intl.formatMessage(blockingMessages.day),
      value: '1440m',
    },
  ];
}
