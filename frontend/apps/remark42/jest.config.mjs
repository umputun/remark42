import path from 'node:path';

/**
 * Plain ESM rather than TypeScript on purpose: a `.ts` config is compiled against
 * `tsconfig.json`, whose `verbatimModuleSyntax` rejects ESM syntax in a file the package has not
 * declared as a module, and which loader jest picks for it varies by environment.
 *
 * @type {import('jest').Config}
 */
const config = {
  testEnvironment: 'jsdom',
  // the dependency-free layer, run by `pnpm test:unit` under plain node. jest is not the runner for
  // those files, and running them here would say nothing about whether they still work with no
  // bundler, which is the property they exist to hold
  testPathIgnorePatterns: ['/node_modules/', '\\.unit\\.test\\.ts$'],
  // babel-jest would otherwise skip node_modules, where .babelrc.js does not reach: passing it as
  // configFile applies the suites' own config to the ESM-only packages below as well
  transform: {
    '^.+\\.(t|j|mj)sx?$': ['babel-jest', { configFile: path.join(import.meta.dirname, '.babelrc.js') }],
  },
  transformIgnorePatterns: ['node_modules/.pnpm/(?!(@testing-library|preact|@github))'],
  moduleDirectories: ['node_modules', 'app'],
  moduleNameMapper: {
    '\\.css': 'identity-obj-proxy',
    // the catalogues are served from the assets embedded in the backend binary
    '^locales/(.*)$': '<rootDir>/../../../backend/app/webassets/assets/$1',
  },
  setupFilesAfterEnv: [
    '<rootDir>/app/__mocks__/fetch.ts',
    '<rootDir>/app/__mocks__/localstorage.ts',
    '<rootDir>/app/__mocks__/resize-observer.ts',
    '<rootDir>/app/__stubs__/remark-config.ts',
    '<rootDir>/app/__stubs__/static-config.ts',
  ],
  collectCoverageFrom: [
    'app/**/*.{js,ts,tsx}',
    // jest excludes a file it ran as a test from coverage, but these it never runs, so without
    // this they are counted as source that nothing covers and drag the whole figure down. their
    // own coverage is reported by the node run in the unit CI job
    '!**/*.unit.test.ts',
    '!**/__mocks__/**',
    '!**/__stubs__/**',
    '!app/tests',
  ],
};

export default config;
