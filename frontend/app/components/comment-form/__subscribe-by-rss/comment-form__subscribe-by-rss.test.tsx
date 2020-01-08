/** @jsx createElement */
import { createElement } from 'preact';
import { shallow } from 'enzyme';

import { SubscribeByRSS, createSubscribeUrl } from './';

jest.mock('react-redux', () => ({
  useSelector: jest.fn(fn => fn({ theme: 'light' })),
}));

describe('<SubscribeByRSS/>', () => {
  let wrapper: ReturnType<typeof shallow>;

  beforeAll(() => {
    wrapper = shallow(<SubscribeByRSS userId="user-1" />);
  });

  it('should be render links in dropdown', () => {
    expect(wrapper.find('.comment-form__rss-dropdown__link')).toHaveLength(3);
  });

  it('should have userId in site link', () => {
    expect(
      wrapper
        .find('.comment-form__rss-dropdown__link')
        .at(1)
        .prop('href')
    ).toEqual(createSubscribeUrl('site', '&user=user-1'));
  });
});
