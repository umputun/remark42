import store from './store';

describe('Store', () => {
  it('should trigger listener on add comment', () => {
    const listener = jest.fn();

    store.set('comments', []);
    store.onUpdate('comments', listener);
    store.addComment({ id: `new` });

    expect(listener).toBeCalledWith([{ comment: { id: 'new' } }]);
  });
});
