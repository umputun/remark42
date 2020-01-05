/** @jsx createElement */
import { createElement, FunctionComponent } from 'preact';

import useTheme from '@app/hooks/useTheme';
import { siteId, url } from '@app/common/settings';
import { BASE_URL, API_BASE } from '@app/common/constants';
import { Dropdown, DropdownItem } from '@app/components/dropdown';

const createSubscribeUrl = (type: 'post' | 'site' | 'reply', urlParams: string = '') =>
  `${BASE_URL}${API_BASE}/rss/${type}?site=${siteId}${urlParams}`;

const items: Array<[string, string]> = [
  [createSubscribeUrl('post'), 'Thread'],
  [createSubscribeUrl('site', '&user='), 'Site'],
  [createSubscribeUrl('reply', `&url=${url}`), 'Replies'],
];

export const SubscribeByRSS: FunctionComponent = () => {
  const theme = useTheme();

  return (
    <Dropdown title="RSS" titleClass="comment-form__rss-dropdown__title" mix="comment-form__rss-dropdown" theme={theme}>
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
