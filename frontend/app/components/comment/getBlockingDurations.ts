import { BlockingDuration } from 'common/types';
import { IntlShape, defineMessages } from 'react-intl';

const blockingMessages = defineMessages({
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
