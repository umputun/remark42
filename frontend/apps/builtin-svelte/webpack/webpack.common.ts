import path from 'path';
import type { Configuration } from 'webpack';
import sveltePreprocess from 'svelte-preprocess';
import TsconfigPathsPlugin from 'tsconfig-paths-webpack-plugin';
import MiniCssExtractPlugin from 'mini-css-extract-plugin';

export const mode = (process.env.NODE_ENV ?? 'development') as 'development' | 'production';
export const port = process.env.PORT || 3000;
export const isProduction = mode === 'production';
export const isDevelopment = !isProduction;
export const apiBaseUrl = process.env.REMARK_API_BASE_URL || 'http://127.0.0.1:8080';
export const htmlMinifyOptions = {
  minifyCSS: true,
  minifyJS: true,
  removeComments: true,
  removeRedundantAttributes: true,
  removeScriptTypeAttributes: true,
  removeStyleLinkTypeAttributes: true,
  sortAttributes: true,
  sortClassName: true,
  useShortDoctype: true,
};

export const commonConfig: Configuration = {
  mode,
  resolve: {
    alias: {
      // Note: Later in this config file, we'll automatically add paths from `tsconfig.compilerOptions.paths`
      svelte: path.resolve('node_modules', 'svelte'),
    },
    extensions: ['.js', '.ts', '.svelte'],
    mainFields: ['svelte', 'browser', 'module', 'main'],
    plugins: [new TsconfigPathsPlugin()],
  },
  output: {
    path: path.resolve(__dirname, 'public'),
    iife: true,
    filename: '[name].js',
    chunkFilename: '[name].[id].js',
    publicPath: '/',
  },
  module: {
    rules: [
      // Rule: Svelte
      {
        test: /\.svelte$/,
        use: [
          {
            loader: isDevelopment ? 'esbuild-loader' : 'babel-loader',
          },
          {
            loader: 'svelte-loader',
            options: {
              compilerOptions: {
                // Dev mode must be enabled for HMR to work!
                dev: isDevelopment,
              },
              emitCss: isProduction,
              hotReload: isDevelopment,
              hotOptions: {
                // List of options and defaults: https://www.npmjs.com/package/svelte-loader-hot#usage
                noPreserveState: false,
                optimistic: true,
              },
              preprocess: sveltePreprocess({
                postcss: true,
              }),
            },
          },
        ],
      },

      {
        // required to prevent errors from Svelte on Webpack 5+, omit on Webpack 4
        test: /node_modules\/svelte\/.*\.mjs$/,
        resolve: {
          fullySpecified: false,
        },
      },

      // Rule: TypeScript
      {
        test: /\.ts$/,
        loader: 'ts-loader',
        options: {
          transpileOnly: true,
        },
      },

      // Rule: CSS
      {
        test: /\.css$/,
        use: [
          {
            loader: MiniCssExtractPlugin.loader,
          },
          'css-loader',
        ],
      },
    ],
  },
  plugins: [
    new MiniCssExtractPlugin({
      filename: '[name].css',
    }),
  ],
  stats: {
    chunks: false,
    chunkModules: false,
    modules: false,
    assets: true,
    entrypoints: false,
  },
};
