import createMockStore from 'redux-mock-store';
import thunk from 'redux-thunk';

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const mockStore = createMockStore<any, any>([thunk]);

export default mockStore;
