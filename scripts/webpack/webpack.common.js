'use strict';

const webpack = require('webpack');
const path = require('path');
const HtmlWebpackPlugin = require('html-webpack-plugin');
const {BundleAnalyzerPlugin} = require('webpack-bundle-analyzer');
const MiniCssExtractPlugin = require('mini-css-extract-plugin');
const { CleanWebpackPlugin } = require('clean-webpack-plugin');
const CopyPlugin = require('copy-webpack-plugin');

module.exports = {
  target: 'web',

  entry: {
    app: './webapp/javascript/index.jsx',
    styles: './webapp/sass/profile.scss',
  },

  output: {
    publicPath: '',
    path: path.resolve(__dirname, '../../webapp/public/build'),
    filename: '[name].[hash].js',
  },

  resolve: {
    extensions: ['.ts', '.tsx', '.es6', '.js', '.jsx', '.json', '.svg'],
    alias: {
      // rc-trigger uses babel-runtime which has internal dependency to core-js@2
      // this alias maps that dependency to core-js@t3
      'core-js/library/fn': 'core-js/stable',
    },
    modules: [
      'node_modules',
      path.resolve('webapp'),
      path.resolve('node_modules'),
    ],
  },

  stats: {
    children: false,
    warningsFilter: /export .* was not found in/,
    source: false,
  },

  watchOptions: {
    ignored: /node_modules/,
  },

  module: {
    // Note: order is bottom-to-top and/or right-to-left
    rules: [
      {
        test: /\.jsx?$/,
        exclude: /node_modules/,
        use: [
          {
            loader: 'babel-loader',
            options: {
              cacheDirectory: true,
              babelrc: true,
              // Note: order is bottom-to-top and/or right-to-left
              presets: [
                [
                  '@babel/preset-env',
                  {
                    targets: {
                      browsers: 'last 3 versions',
                    },
                    useBuiltIns: 'entry',
                    corejs: 3,
                    modules: false,
                  },
                ],
                [
                  '@babel/preset-typescript',
                  {
                    allowNamespaces: true,
                  },
                ],
                '@babel/preset-react',
              ],
            },
          },
        ],
      },
      {
        test: /\.js$/,
        use: [
          {
            loader: 'babel-loader',
            options: {
              presets: [['@babel/preset-env']],
            },
          },
        ],
      },
      {
        test: /\.html$/,
        exclude: /(index|error)\.html/,
        use: [
          {
            loader: 'html-loader',
            options: {
              attrs: [],
              minimize: true,
              removeComments: false,
              collapseWhitespace: false,
            },
          },
        ],
      },
      {
        test: /\.css$/,
        // include: MONACO_DIR, // https://github.com/react-monaco-editor/react-monaco-editor
        use: ['style-loader', 'css-loader'],
      },
      {
        test: /\.scss$/,
        use: [
          MiniCssExtractPlugin.loader,
          {
            loader: 'css-loader',
            options: {
              importLoaders: 2,
              url: true,
              sourceMap: true
            },
          },
          {
            loader: 'postcss-loader',
            options: {
              sourceMap: true,
              config: { path: __dirname },
            },
          },
          {
            loader: 'sass-loader',
            options: {
              sourceMap: true
            },
          },
        ],
      },
      {
        test: /\.(svg|ico|jpg|jpeg|png|gif|eot|otf|webp|ttf|woff|woff2|cur|ani|pdf)(\?.*)?$/,
        loader: 'file-loader',
        options: { name: 'static/img/[name].[hash:8].[ext]' },
      },
    ],
  },

  plugins: [
    new webpack.ProvidePlugin({
      $: 'jquery',
      jQuery: 'jquery',
    }),
    new HtmlWebpackPlugin({
      filename: path.resolve(__dirname, '../../webapp/public/index.html'),
      template: path.resolve(__dirname, '../../webapp/templates/index.html'),
      inject: false,
      chunksSortMode: 'none',
      templateParameters: (compilation, assets, options) => {
        return ({
          mode: process.env.NODE_ENV,
          webpack: compilation.getStats().toJson(),
          compilation: compilation,
          webpackConfig: compilation.options,
          htmlWebpackPlugin: {
            files: assets,
            options: options
          }
        })
      },
    }),
    new MiniCssExtractPlugin({
      filename: '[name].[hash].css',
    }),
    // new BundleAnalyzerPlugin({
    //   analyzerPort: 8889
    // }),
    new webpack.IgnorePlugin(/^\.\/locale$/, /moment$/),
    new CopyPlugin({
      patterns: [
        {
          from: 'webapp/images',
          to: 'images'
        }
      ]
    }),
  ],
};
