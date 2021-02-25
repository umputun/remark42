import capitalizeFirstLetter from './capitalize-first-letter';

it('should capitalize first letter', () => {
  expect(capitalizeFirstLetter('one')).toBe('One');
  expect(capitalizeFirstLetter('один')).toBe('Один');
  expect(capitalizeFirstLetter('用户名最少需要3个字符')).toBe('用户名最少需要3个字符');
});
