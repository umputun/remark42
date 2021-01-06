import type { User } from 'common/types';

const user: Readonly<User> = {
  id: 'email_1',
  name: 'John',
  picture: 'some_picture',
  admin: false,
  ip: '127.0.0.1',
  block: false,
  verified: false,
  email_subscription: false,
};

const anonymousUser: Readonly<User> = {
  ...user,
  id: 'anonymous_1',
};

export { user, anonymousUser };
