import { h, render } from 'preact';
import Preloader from '../preloader';
import {createDomContainer} from 'testUtils';

describe(`<Preloader />`, () => {
  let container;

  createDomContainer(({domContainer}) => {
    container = domContainer;
  });

  it('should render Preloader', () => {
    render(<Preloader mix="root__preloader" />, container);

    expect(container.children[0].className).toEqual('preloader root__preloader');
  });
});
