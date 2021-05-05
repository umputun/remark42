import createMockStore from 'redux-mock-store';
import thunk from 'redux-thunk';

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export const mockStore = createMockStore<any, any>([thunk]);
