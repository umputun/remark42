jest.mock('react-redux', () => ({
  useSelector: jest.fn(fn => fn()),
}));

import { useSelector as useSelectorToMock } from 'react-redux';

export const useSelector = useSelectorToMock as jest.Mock<any>; // eslint-disable-line @typescript-eslint/no-explicit-any
