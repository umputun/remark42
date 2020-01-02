/** @jsx createElement */
import { createElement } from 'preact';
import { shallow } from 'enzyme';

import { CommentForm, Props } from './comment-form';

describe('<CommentForm />', () => {
  it('shoud render without control panel, preview button, and rss links in "simple view" mode', () => {
    const element = shallow(<CommentForm {...({ simpleView: true } as Props)} />);

    expect(element.exists('.comment-form__control-panel')).toEqual(false);
    expect(element.exists('.comment-form__button_type_preview')).toEqual(false);
    expect(element.exists('.comment-form__rss')).toEqual(false);
  });
});
