import { h, FunctionComponent } from 'preact';
import { useMemo } from 'preact/hooks';
import { useIntl, defineMessages } from 'react-intl';

import { siteId, url } from 'common/settings';
import { BASE_URL, API_BASE } from 'common/constants';
import { Dropdown, DropdownItem } from 'components/dropdown';

import styles from './subscribe-by-rss.module.css';
import { useTheme } from 'hooks/useTheme';

export const SubscribeByRSS: FunctionComponent<{ userId: string | undefined }> = ({ userId }) => {
  const intl = useIntl();
  const theme = useTheme();
  const items: Array<[string, string]> = useMemo(
    () => [
      [createSubscribeUrl('post', `&url=${url}`), intl.formatMessage(messages.thread)],
      [createSubscribeUrl('site'), intl.formatMessage(messages.site)],
      [createSubscribeUrl('reply', `&user=${userId}`), intl.formatMessage(messages.replies)],
    ],
    [userId, intl]
  );

  return (
    <Dropdown
      theme={theme}
      title={intl.formatMessage(messages.title)}
      titleClass={styles.title}
      buttonTitle={intl.formatMessage(messages.buttonTitle)}
    >
      {items.map(([href, label]) => (
        <DropdownItem key={label}>
          <a href={href} className={styles.link} target="_blank" rel="noreferrer">
            {label}
          </a>
        </DropdownItem>
      ))}
    </Dropdown>
  );
};

export const createSubscribeUrl = (type: 'post' | 'site' | 'reply', urlParams = '') =>
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
