/* eslint-disable no-console */
const path = require('path');

const webpack = require('webpack');
const ExtractText = require('extract-text-webpack-plugin');
const Clean = require('clean-webpack-plugin');
const Copy = require('copy-webpack-plugin');
const Html = require('html-webpack-plugin');
const Provide = webpack.ProvidePlugin;
const Define = webpack.DefinePlugin;
const BundleAnalyze = require('webpack-bundle-analyzer').BundleAnalyzerPlugin;

const babelOptions = require('./babelOptions');
const { NODE_ID } = require('./app/common/constants');
const publicFolder = path.resolve(__dirname, 'public');
const env = process.env.NODE_ENV || 'dev';

// let's log some env variables because we can
console.log(`NODE_ENV = ${env}`);

const commonStyleLoaders = [
  'css-loader',
  {
    loader: 'postcss-loader',
    options: {
      plugins: [
        require('autoprefixer')({ browsers: ['> 1%'] }),
        require('postcss-url')({ url: 'inline', maxSize: 5 }),
        require('postcss-wrap')({ selector: `#${NODE_ID}` }),
        require('postcss-csso'),
      ]
    }
  },
  {
    loader: 'sass-loader',
    options: {
      includePaths: [path.resolve(__dirname, 'app')],
    }
  }
];

module.exports = {
  context: __dirname,
  entry: {
    embed: './app/embed',
    counter: './app/counter',
    'last-comments': './app/last-comments',
    remark: './app/remark',
  },
  output: {
    path: publicFolder,
    filename: `[name].js`,
    chunkFilename: '[name].js'
  },
  resolve: {
    extensions: ['.jsx', '.js'],
    modules: [path.resolve(__dirname, 'app'), path.resolve(__dirname, 'node_modules')]
  },
  devtool: 'source-map',
  module: {
    rules: [
      {
        test: /\.jsx?$/,
        exclude: /(node_modules|\.vendor\.js$)/,
        use: {
          loader: 'babel-loader',
          options: babelOptions,
        }
      },
      {
        test: /\.scss$/,
        use: env === 'production'
          ? ExtractText.extract({
            fallback: 'style-loader',
            use: commonStyleLoaders,
          })
          : [
            'style-loader',
            ...commonStyleLoaders,
          ],
      },
      {
        test: /\.(png|jpg|jpeg|gif|svg)$/,
        use: {
          loader: 'file-loader',
          options: {
            name: `files/[name].[hash].[ext]`
          }
        }
      }
    ]
  },
  plugins: [
    new Clean(publicFolder),
    new Provide({
      b: 'bem-react-helper',
    }),
    new Define({
      'process.env.NODE_ENV': JSON.stringify(process.env.NODE_ENV),
    }),
    // TODO: we should add it only on demo serv
    new Html({
      template: path.resolve(__dirname, 'index.ejs'),
      inject: false,
    }),
    // TODO: we should add it only on demo serv
    new Html({
      template: path.resolve(__dirname, 'counter.ejs'),
      filename: 'counter.html',
      inject: false,
    }),
    // TODO: we should add it only on demo serv
    new Html({
      template: path.resolve(__dirname, 'last-comments.ejs'),
      filename: 'last-comments.html',
      inject: false,
    }),
    ...(env === 'production' ? [] : [new Html({
      template: path.resolve(__dirname, 'dev.ejs'),
      filename: 'dev.html',
      inject: false,
    })]),
    new ExtractText({
      filename: `remark.css`,
      allChunks: true
    }),
    new webpack.optimize.ModuleConcatenationPlugin(),
    ...(env === 'production' ? [new webpack.optimize.UglifyJsPlugin()] : []),
    ...(process.env.CI === 'true' ? [] : [
      new BundleAnalyze({
        analyzerMode: 'static',
        reportFilename: 'report.html',
        defaultSizes: 'parsed',
        generateStatsFile: false,
        logLevel: 'info',
        openAnalyzer: false
      })
    ]),
    new Copy(['./iframe.html']),
  ],
  watch: env === 'dev',
  watchOptions: {
    ignored: /(node_modules|\.vendor\.js$)/
  },
  devServer: {
    host: '0.0.0.0',
    port: 8080,
    contentBase: publicFolder,
  },
};
