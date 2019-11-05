/** @jsx createElement */
import { createElement } from 'preact';
import { mount } from 'enzyme';
import Preloader from './preloader';

describe(`<Preloader />`, () => {
  it('should render Preloader', () => {
    const element = mount(<Preloader mix="root__preloader" />);

    expect(element.childAt(0).hasClass('preloader root__preloader')).toEqual(true);
  });
});
