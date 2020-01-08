/** @jsx createElement */
import { createElement } from 'preact';
import { shallow } from 'enzyme';

import { user } from '@app/testUtils/mocks/user';
import { StaticStore } from '@app/common/static_store';

import { CommentForm, Props } from './comment-form';
import { SubscribeByEmail } from './__subscribe-by-email';

const DEFAULT_PROPS: Readonly<Props> = {
  mode: 'main',
  theme: 'light',
  onSubmit: () => Promise.resolve(),
  getPreview: () => Promise.resolve(''),
  user: null,
};

describe('<CommentForm />', () => {
  it('should render without control panel, preview button, and rss links in "simple view" mode', () => {
    const props = { ...DEFAULT_PROPS, simpleView: true };
    const wrapper = shallow(<CommentForm {...props} />);

    expect(wrapper.exists('.comment-form__control-panel')).toEqual(false);
    expect(wrapper.exists('.comment-form__button_type_preview')).toEqual(false);
    expect(wrapper.exists('.comment-form__rss')).toEqual(false);
  });

  it('should be rendered with email subscription button', () => {
    StaticStore.config.email_notifications = true;

    const props = { ...DEFAULT_PROPS, user };
    const wrapper = shallow(<CommentForm {...props} />);

    expect(wrapper.exists(SubscribeByEmail)).toEqual(true);
  });

  it('should be rendered without email subscription button when email_notifications disabled', () => {
    StaticStore.config.email_notifications = false;

    const props = { ...DEFAULT_PROPS, user };
    const wrapper = shallow(<CommentForm {...props} />);

    expect(wrapper.exists(SubscribeByEmail)).toEqual(false);
  });
});
