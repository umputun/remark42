/** @jsx h */
import { h, render } from 'preact';
import Preloader from './preloader';
import { createDomContainer } from '@app/testUtils';

describe(`<Preloader />`, () => {
  let container: HTMLElement;

  createDomContainer(domContainer => {
    container = domContainer;
  });

  it('should render Preloader', () => {
    render(<Preloader mix="root__preloader" />, container);

    expect(container.children[0].className).toEqual('preloader root__preloader');
  });
});
