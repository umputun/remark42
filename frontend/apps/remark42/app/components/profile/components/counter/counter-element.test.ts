import { counterElement } from './counter-element.js';

describe('the counter', () => {
  it('shows the count', () => {
    expect(counterElement(11).textContent).toBe('11');
  });
});
