import { h } from 'preact';
import { useMemo } from 'preact/hooks';
import { useIntl, defineMessages } from 'react-intl';

import { siteId, url } from 'common/settings';
import { BASE_URL, API_BASE } from 'common/constants';
import { Dropdown } from 'components/ui/dropdown';
import { IconButton } from 'components/icon-button/icon-button';
import { RssIcon } from 'components/icons/rss';
import { Tooltip } from 'components/ui/tooltip';

export const createSubscribeUrl = (type: 'post' | 'site' | 'reply', urlParams = '') =>
  `${BASE_URL}${API_BASE}/rss/${type}?site=${siteId}${urlParams}`;

type Props = {
  userId: string | undefined;
};

export function SubscribeByRSS({ userId }: Props) {
  const intl = useIntl();
  const items = useMemo((): [string, string][] => {
    const list: [string, string][] = [
      [createSubscribeUrl('post', `&url=${url}`), intl.formatMessage(messages.thread)],
      [createSubscribeUrl('site'), intl.formatMessage(messages.site)],
    ];

    if (userId) {
      list.push([createSubscribeUrl('reply', `&user=${userId}`), intl.formatMessage(messages.replies)]);
    }

    return list;
  }, [userId, intl]);

  return (
    <Dropdown
      position="bottom-left"
      button={
        <IconButton>
            <RssIcon size={20} />
          {/* <Tooltip text="Subscribe on RSS feed" position="bottom-left">
          </Tooltip> */}
        </IconButton>
      }
    >
      {items.map(([href, label]) => (
        <a key={href} href={href} target="_blank" rel="noreferrer">
          {label}
        </a>
      ))}
    </Dropdown>
  );
}

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
