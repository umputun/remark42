/** @jsx createElement */
import { createElement } from 'preact';
import { shallow } from 'enzyme';

import { Input, Props } from './input';

describe('<Input />', () => {
  it('shoud render without control panel, preview button, and rss links in "simple view" mode', () => {
    const element = shallow(<Input {...({ simpleView: true } as Props)} />);

    expect(element.exists('.input__control-panel')).toEqual(false);
    expect(element.exists('.input__button_type_preview')).toEqual(false);
    expect(element.exists('.input__rss')).toEqual(false);
  });
});
