/** @jsx createElement */
import { createElement, FunctionComponent } from 'preact';
import { useMemo } from 'preact/hooks';
import { useIntl, defineMessages } from 'react-intl';

import useTheme from '@app/hooks/useTheme';
import { siteId, url } from '@app/common/settings';
import { BASE_URL, API_BASE } from '@app/common/constants';
import { Dropdown, DropdownItem } from '@app/components/dropdown';

export const createSubscribeUrl = (type: 'post' | 'site' | 'reply', urlParams: string = '') =>
  `${BASE_URL}${API_BASE}/rss/${type}?site=${siteId}${urlParams}`;

const messages = defineMessages({
  thread: {
    id: 'subscribeByRSS.thread',
    defaultMessage: 'Thread',
  },
  site: {
    id: 'subscribeByRSS.site',
    defaultMessage: 'Site',
  },
  replies: {
    id: 'subscribeByRSS.replies',
    defaultMessage: 'Replies',
  },
  buttonTitle: {
    id: 'subscribeByRSS.button-title',
    defaultMessage: 'Subscribe by RSS',
  },
  title: {
    id: 'subscribeByRSS.title',
    defaultMessage: 'RSS',
  },
});

export const SubscribeByRSS: FunctionComponent<{ userId: string | null }> = ({ userId }) => {
  const theme = useTheme();
  const intl = useIntl();
  const items: Array<[string, string]> = useMemo(
    () => [
      [createSubscribeUrl('post'), intl.formatMessage(messages.thread)],
      [createSubscribeUrl('site', `&user=${userId}`), intl.formatMessage(messages.site)],
      [createSubscribeUrl('reply', `&url=${url}`), intl.formatMessage(messages.replies)],
    ],
    [userId]
  );

  return (
    <Dropdown
      title={intl.formatMessage(messages.title)}
      titleClass="comment-form__rss-dropdown__title"
      buttonTitle={intl.formatMessage(messages.buttonTitle)}
      mix="comment-form__rss-dropdown"
      theme={theme}
    >
      {items.map(([href, label]) => (
        <DropdownItem>
          <a href={href} className="comment-form__rss-dropdown__link" target="_blank">
            {label}
          </a>
        </DropdownItem>
      ))}
    </Dropdown>
  );
};
