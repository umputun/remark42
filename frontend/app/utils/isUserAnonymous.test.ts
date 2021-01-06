import { isUserAnonymous } from './isUserAnonymous';
import { User } from 'common/types';

describe('isUserAnonymous', () => {
  test('user is anonymous', () => {
    expect(isUserAnonymous(null)).toEqual(true);
    expect(isUserAnonymous({ id: 'anonymous_1' } as User)).toEqual(true);
  });

  test('user is not anonymous', () => {
    expect(isUserAnonymous({ id: 'email_1' } as User)).toEqual(false);
  });
});
