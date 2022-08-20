import path from 'path';
import merge from 'webpack-merge';
import HtmlWebpackPlugin from 'html-webpack-plugin';

import { commonConfig, htmlMinifyOptions, isDevelopment } from './webpack.common';

const legacyConfig = merge([
  commonConfig,
  {
    target: 'browserslist:legacy',
    entry: {
      'config': './src/modules/config.ts',
      'inject': './src/modules/load-module.ts',
      'load-module': './src/modules/load-module.ts',
      'comments': './src/modules/comments.ts',
    },
    plugins: [
      new HtmlWebpackPlugin({
        template: path.resolve(__dirname, 'src/pages/demo.ejs'),
        filename: 'index.html',
        inject: false,
        minify: htmlMinifyOptions,
      }),
      new HtmlWebpackPlugin({
        template: path.resolve(__dirname, 'src/pages/comments.ejs'),
        filename: 'comments.html',
        inject: isDevelopment,
        chunks: ['comments'],
        minify: htmlMinifyOptions,
      }),
    ],
  },
]);

const modernConfig = merge([
  commonConfig,
  {
    target: 'browserslist:modern',
    entry: {
      comments: './src/modules/comments.ts',
    },
    output: {
      filename: '[name].mjs',
      chunkFilename: '[name].mjs',
    },
  },
]);

export const prodConfig = [legacyConfig, modernConfig];
