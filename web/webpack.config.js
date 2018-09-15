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
const publicFolder = path.resolve(__dirname, 'public');
const env = process.env.NODE_ENV || 'development';
const remarkUrl = process.env.REMARK_URL || 'https://demo.remark42.com';
const NODE_ID = 'remark42';
// let's log some env variables because we can
console.log(`NODE_ENV = ${env}`);
console.log(`REMARK_ENV = ${remarkUrl}`);

const commonStyleLoaders = [
  'css-loader',
  {
    loader: 'postcss-loader',
    options: {
      plugins: [
        require('postcss-for'),
        require('postcss-simple-vars'),
        require('postcss-nested'),
        require('postcss-calc'),
        require('autoprefixer')({ browsers: ['> 1%'] }),
        require('postcss-url')({ url: 'inline', maxSize: 5 }),
        require('postcss-wrap')({ selector: `#${NODE_ID}` }),
        require('postcss-csso'),
      ],
    },
  },
];

module.exports = {
  context: __dirname,
  devtool: env === 'development' ? 'source-map' : false,
  entry: {
    embed: './app/embed',
    counter: './app/counter',
    'last-comments': './app/last-comments',
    remark: './app/remark',
    deleteme: './app/deleteme',
  },
  output: {
    path: publicFolder,
    filename: `[name].js`,
    chunkFilename: '[name].js',
  },
  resolve: {
    extensions: ['.jsx', '.js'],
    modules: [path.resolve(__dirname, 'app'), path.resolve(__dirname, 'node_modules')],
  },
  module: {
    rules: [
      {
        test: /\.jsx?$/,
        exclude: /(node_modules|\.vendor\.js$)/,
        use: {
          loader: 'babel-loader',
          options: babelOptions,
        },
      },
      {
        test: /\.scss$/,
        use: ExtractText.extract({
          fallback: 'style-loader',
          use: commonStyleLoaders,
        }),
      },
      {
        test: /\.(png|jpg|jpeg|gif|svg)$/,
        use: {
          loader: 'file-loader',
          options: {
            name: `files/[name].[hash].[ext]`,
          },
        },
      },
    ],
  },
  plugins: [
    new Clean(publicFolder),
    new Provide({
      b: 'bem-react-helper',
    }),
    new Define({
      'process.env.NODE_ENV': JSON.stringify(env),
      'process.env.REMARK_NODE': JSON.stringify(NODE_ID),
      'process.env.REMARK_URL': env === 'production' ? JSON.stringify(remarkUrl) : 'window.location.origin',
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
    new ExtractText({
      filename: `remark.css`,
      allChunks: true,
    }),
    new webpack.optimize.ModuleConcatenationPlugin(),
    ...(env === 'production' ? [new webpack.optimize.UglifyJsPlugin()] : []),
    ...(process.env.CI
      ? []
      : [
          new BundleAnalyze({
            analyzerMode: 'static',
            reportFilename: 'report.html',
            defaultSizes: 'parsed',
            generateStatsFile: false,
            logLevel: 'info',
            openAnalyzer: false,
          }),
        ]),
    new Copy(['./iframe.html', './deleteme.html', './markdown-help.html']),
  ],
  watchOptions: {
    ignored: /(node_modules|\.vendor\.js$)/,
  },
  devServer: {
    host: 'localhost',
    port: 9000,
    contentBase: publicFolder,
    publicPath: '/web',
    proxy: {
      '/api': {
        target: remarkUrl,
        logLevel: 'debug',
        changeOrigin: true,
      },
      '/auth': {
        target: remarkUrl,
        logLevel: 'debug',
        changeOrigin: true,
      },
    },
  },
};
