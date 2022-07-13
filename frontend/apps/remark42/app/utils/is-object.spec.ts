import { isObject } from './is-object';

describe('is-object', () => {
  it.each`
    input
    ${null}
    ${undefined}
    ${0}
    ${1}
    ${''}
    ${'string'}
    ${true}
    ${false}
    ${[]}
    ${[1, 2, 3]}
  `('should return false if is NOT an object', ({ input }) => {
    expect(isObject(input)).toBe(false);
  });
  it.each`
    input
    ${{}}
    ${{ a: 1 }}
    ${{ a: 1, b: 2 }}
    ${new Error()}
  `('should return true if IS an object', ({ input }) => {
    expect(isObject(input)).toBe(true);
  });
});
