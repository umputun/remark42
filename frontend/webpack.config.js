/* eslint-disable no-console */

const path = require('path');

const webpack = require('webpack');
const MiniCssExtractPlugin = require('mini-css-extract-plugin');
const { CleanWebpackPlugin } = require('clean-webpack-plugin');
const Copy = require('copy-webpack-plugin');
const Html = require('html-webpack-plugin');
const Define = webpack.DefinePlugin;
const BundleAnalyze = require('webpack-bundle-analyzer').BundleAnalyzerPlugin;

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
        require('autoprefixer')({ overrideBrowserslist: ['> 1%'] }),
        require('postcss-url')({ url: 'inline', maxSize: 5 }),
        require('postcss-wrap')({ selector: `#${NODE_ID}` }),
        require('postcss-csso'),
      ],
    },
  },
];

module.exports = () => ({
  context: __dirname,
  devtool: env === 'development' ? 'source-map' : false,
  entry: {
    embed: './app/embed.ts',
    counter: './app/counter.ts',
    'last-comments': './app/last-comments.tsx',
    remark: './app/remark.tsx',
    deleteme: './app/deleteme.ts',
  },
  output: {
    path: publicFolder,
    filename: `[name].js`,
    chunkFilename: '[name].js',
  },
  resolve: {
    extensions: ['.tsx', '.ts', '.jsx', '.js'],
    alias: { '@app': path.resolve(__dirname, 'app') },
    modules: [path.resolve(__dirname, 'node_modules')],
  },
  module: {
    rules: [
      {
        test: /\.js(x?)$/,
        oneOf: [
          {
            include: /node_modules\/@github\/markdown-toolbar-element/,
            use: {
              loader: 'babel-loader',
              options: {
                presets: [
                  [
                    '@babel/preset-env',
                    {
                      targets: {
                        browsers: ['> 1%', 'android >= 4.4.4', 'ios >= 9', 'IE >= 11'],
                      },
                      useBuiltIns: 'usage',
                      corejs: 3,
                    },
                  ],
                ],
              },
            },
          },
          {
            exclude: /node_modules/,
            use: 'babel-loader',
          },
        ],
      },
      {
        test: /\.ts(x?)$/,
        exclude: /node_modules/,
        use: ['babel-loader', 'ts-loader'],
      },
      {
        test: /\.s?css$/,
        use: [
          {
            loader: MiniCssExtractPlugin.loader,
          },
          ...commonStyleLoaders,
        ],
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
    new CleanWebpackPlugin(),
    new Define({
      'process.env.NODE_ENV': JSON.stringify(env),
      'process.env.REMARK_NODE': JSON.stringify(NODE_ID),
      'process.env.REMARK_URL': env === 'production' ? JSON.stringify(remarkUrl) : 'window.location.origin',
    }),
    new Html({
      template: path.resolve(__dirname, 'index.ejs'),
      inject: false,
    }),
    new Html({
      template: path.resolve(__dirname, 'counter.ejs'),
      filename: 'counter.html',
      inject: false,
    }),
    new Html({
      template: path.resolve(__dirname, 'last-comments.ejs'),
      filename: 'last-comments.html',
      inject: false,
    }),
    new Html({
      template: path.resolve(__dirname, 'comments.ejs'),
      filename: 'comments.html',
      inject: false,
    }),
    new MiniCssExtractPlugin({
      filename: '[name].css',
    }),
    new webpack.optimize.ModuleConcatenationPlugin(),
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
    aggregateTimeout: 3000,
  },
  stats: {
    children: false,
    entrypoints: false,
  },
  devServer: {
    host: '0.0.0.0',
    port: 9000,
    contentBase: publicFolder,
    publicPath: '/web',
    disableHostCheck: true,
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
    stats: {
      children: false,
      entrypoints: false,
    },
  },
});
