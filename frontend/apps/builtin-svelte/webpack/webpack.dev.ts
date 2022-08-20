import path from 'path';
import { merge } from 'webpack-merge';
import HtmlWebpackPlugin from 'html-webpack-plugin';

import { commonConfig, isDevelopment, htmlMinifyOptions, port, apiBaseUrl as target } from './webpack.common';

export const devConfig = merge([
  commonConfig,
  {
    target: 'browserslist:modern',
    entry: {
      // embeders
      'embed-comments': './src/embeders/comments.ts',

      // entries
      'comments': './src/entries/comments.ts',
      'last-comments': './src/entries/last-comments.ts',

      // injectables
      'config': './src/injectables/config.ts',
      'load-module': './src/injectables/load-module.ts',
    },
    output: {
      filename: '[name].js',
      chunkFilename: '[name].js',
    },
    plugins: [
      new HtmlWebpackPlugin({
        template: path.resolve(__dirname, '../src/pages/demo.ejs'),
        filename: 'index.html',
        inject: false,
        minify: htmlMinifyOptions,
      }),
      new HtmlWebpackPlugin({
        template: path.resolve(__dirname, '../src/pages/comments.ejs'),
        filename: 'comments.html',
        minify: htmlMinifyOptions,
        inject: false,
      }),
    ],
    devtool: isDevelopment ? 'inline-cheap-source-map' : false,
    devServer: {
      port,
      devMiddleware: {
        stats: 'minimal',
      },
      headers: {
        'Access-Control-Allow-Origin': '*',
        'Access-Control-Allow-Methods': 'GET, POST, PUT, DELETE, PATCH, OPTIONS',
        'Access-Control-Allow-Headers': 'X-Requested-With, content-type, Authorization',
      },
      allowedHosts: 'all',
      proxy: [
        { path: '/api', target, changeOrigin: true },
        { path: '/auth', target, changeOrigin: true },
      ],
    },
  },
]);
