/** @jsx createElement */
import { createElement, FunctionComponent } from 'preact';
import { useMemo } from 'preact/hooks';

import useTheme from '@app/hooks/useTheme';
import { siteId, url } from '@app/common/settings';
import { BASE_URL, API_BASE } from '@app/common/constants';
import { Dropdown, DropdownItem } from '@app/components/dropdown';

export const createSubscribeUrl = (type: 'post' | 'site' | 'reply', urlParams: string = '') =>
  `${BASE_URL}${API_BASE}/rss/${type}?site=${siteId}${urlParams}`;

export const SubscribeByRSS: FunctionComponent<{ userId: string | null }> = ({ userId }) => {
  const theme = useTheme();
  const items: Array<[string, string]> = useMemo(
    () => [
      [createSubscribeUrl('post'), 'Thread'],
      [createSubscribeUrl('site', `&user=${userId}`), 'Site'],
      [createSubscribeUrl('reply', `&url=${url}`), 'Replies'],
    ],
    [userId]
  );

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
