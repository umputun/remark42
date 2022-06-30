import type { Config } from 'jest';

const config: Config = {
  testEnvironment: 'jsdom',
  transform: {
    '^.+\\.(t|j|mj)sx?$': [
      '@swc/jest',
      {
        jsc: {
          parser: {
            syntax: 'typescript',
            tsx: true,
            decorators: false,
          },
          target: 'es2016',
          transform: {
            react: {
              runtime: 'automatic',
              importSource: 'preact',
            },
          },
        },
      },
    ],
  },
  transformIgnorePatterns: ['node_modules/(?!(@testing-library/preact|preact|@github|lodash-es))'],
  moduleDirectories: ['node_modules', 'app'],
  moduleNameMapper: {
    '\\.css': 'identity-obj-proxy',
    '\\.svg': '<rootDir>/app/__stubs__/svg.tsx',
    '^react$': 'preact/compat',
    '^react-dom$': 'preact/compat',
  },
  setupFiles: ['<rootDir>/jest.setup.ts'],
  setupFilesAfterEnv: [
    '<rootDir>/app/__mocks__/fetch.ts',
    '<rootDir>/app/__mocks__/localstorage.ts',
    '<rootDir>/app/__stubs__/remark-config.ts',
    '<rootDir>/app/__stubs__/static-config.ts',
  ],
  collectCoverageFrom: [
    'app/**/*.{ts,tsx}',
    '!**/__mocks__/**',
    '!**/__stubs__/**',
    '!app/locales/**',
    '!app/utils/loadLocale.ts',
    '!app/tests',
  ],
};

export default config;
