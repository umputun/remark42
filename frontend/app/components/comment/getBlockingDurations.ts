import { BlockingDuration } from '@app/common/types';
import { IntlShape, defineMessages } from 'react-intl';

defineMessages({
  'blockingDuration.permanently': {
    id: 'blockingDuration.permanently',
    defaultMessage: 'Permanently',
  },
  'blockingDuration.month': {
    id: 'blockingDuration.month',
    defaultMessage: 'For a month',
  },
  'blockingDuration.week': {
    id: 'blockingDuration.week',
    defaultMessage: 'For a week',
  },
  'blockingDuration.day': {
    id: 'blockingDuration.day',
    defaultMessage: 'For a day',
  },
});

export function getBlockingDurations(intl: IntlShape): BlockingDuration[] {
  return [
    {
      label: intl.formatMessage({
        id: 'blockingDuration.permanently',
        defaultMessage: 'blockingDuration.permanently',
      }),
      value: 'permanently',
    },
    {
      label: intl.formatMessage({
        id: 'blockingDuration.month',
        defaultMessage: 'blockingDuration.month',
      }),
      value: '43200m',
    },
    {
      label: intl.formatMessage({
        id: 'blockingDuration.week',
        defaultMessage: 'blockingDuration.week',
      }),
      value: '10080m',
    },
    {
      label: intl.formatMessage({
        id: 'blockingDuration.day',
        defaultMessage: 'blockingDuration.day',
      }),
      value: '1440m',
    },
  ];
}
