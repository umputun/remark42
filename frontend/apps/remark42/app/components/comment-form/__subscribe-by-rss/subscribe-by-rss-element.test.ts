import en from 'locales/en.json';
import { createIntl } from 'common/intl-message';

import { createSubscribeUrl, mountSubscribeByRss } from './subscribe-by-rss-element.js';

const intl = createIntl('en', en);

afterEach(() => {
  document.body.replaceChildren();
});

describe('mountSubscribeByRss', () => {
  it('offers the replies of the reader the latest props name, from the next opening', () => {
    const root = document.createElement('div');
    document.body.append(root);
    const rss = mountSubscribeByRss(root, { userId: 'first', theme: 'light' }, intl);
    const title = root.querySelector('button') as HTMLButtonElement;
    const replies = () => root.querySelectorAll('a')[2]?.getAttribute('href');

    title.click();
    expect(replies()).toBe(createSubscribeUrl('reply', '&user=first'));
    title.click();

    rss.update({ userId: 'second', theme: 'light' });
    title.click();
    expect(replies()).toBe(createSubscribeUrl('reply', '&user=second'));
    rss.destroy();
  });
});
