module.exports = {
  roots: ['<rootDir>/app/'],
  transform: {
    '^.+\\.ts(x?)$': 'ts-jest',
    '^.+\\.js(x?)$': 'babel-jest',
  },
  testMatch: null,
  testRegex: '(\\.|/)(test|spec)\\.(j|t)s(x?)$',
  moduleFileExtensions: ['ts', 'tsx', 'js', 'jsx', 'json', 'node'],
  moduleNameMapper: {
    '\\.scss$': '<rootDir>/app/testUtils/mockStyles.js',
    '@app/(.*)': '<rootDir>/app/$1',
  },
  setupFilesAfterEnv: ['<rootDir>/app/testUtils/index.ts'],
};
