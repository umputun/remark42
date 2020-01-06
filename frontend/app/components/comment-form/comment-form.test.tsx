/** @jsx createElement */
import { createElement } from 'preact';
import { shallow } from 'enzyme';

import { StaticStore } from '@app/common/static_store';

import { CommentForm, Props } from './comment-form';

const DEFAULT_PROPS: Readonly<Props> = {
  mode: 'main',
  theme: 'light',
  onSubmit: () => Promise.resolve(),
  getPreview: () => Promise.resolve(''),
};

describe('<CommentForm />', () => {
  it('should render without control panel, preview button, and rss links in "simple view" mode', () => {
    const props = { ...DEFAULT_PROPS, simpleView: true };
    const element = shallow(<CommentForm {...props} />);

    expect(element.exists('.comment-form__control-panel')).toEqual(false);
    expect(element.exists('.comment-form__button_type_preview')).toEqual(false);
    expect(element.exists('.comment-form__rss')).toEqual(false);
  });

  it('should render with email subscription button', () => {
    StaticStore.config.email_notifications = true;

    const element = shallow(<CommentForm {...DEFAULT_PROPS} />);
    const emailDropdown = element.find({ mix: 'comment-form__email-dropdown' });

    expect(emailDropdown).toHaveLength(1);
  });

  it('should render without email subscription button', () => {
    StaticStore.config.email_notifications = false;

    const element = shallow(<CommentForm {...DEFAULT_PROPS} />);
    const emailDropdown = element.find({ mix: 'comment-form__email-dropdown' });

    expect(emailDropdown).toHaveLength(0);
  });
});
