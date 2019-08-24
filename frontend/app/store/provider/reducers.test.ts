import reducer from './reducers';
import { PROVIDER_UPDATE } from './types';

describe('provider reducer', () => {
  it('should set name of provider', () => {
    const result = reducer.provider(
      { name: null },
      {
        type: PROVIDER_UPDATE,
        payload: {
          name: 'something',
        },
      }
    );
    expect(result).toStrictEqual({
      name: 'something',
    });
  });
});
