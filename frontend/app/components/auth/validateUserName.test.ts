import { validateUserName } from './validateUserName';

describe('validate user name', () => {
  it('should allow good name', () => {
    expect(validateUserName('Раз_Два Три_34567')).toEqual(true);
  });
  it('should not allow bad name', () => {
    expect(validateUserName('**blah123')).toEqual(false);
  });
  it('should not allow only spaces', () => {
    expect(validateUserName('     ')).toEqual(false);
  });
});
