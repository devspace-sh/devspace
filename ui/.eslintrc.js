module.exports = {
  root: true,
  env: {
    browser: true,
    es6: true,
    node: true,
  },
  parser: '@typescript-eslint/parser',
  parserOptions: {
    ecmaFeatures: {
      jsx: true,
    },
    ecmaVersion: 2019,
    sourceType: 'module',
  },
  plugins: ['@typescript-eslint', 'react'],
  extends: ['eslint:recommended', 'plugin:react/recommended'],
  settings: {
    react: {
      version: 'detect',
    },
  },
  rules: {
    '@typescript-eslint/no-unused-vars': 'off',
    'no-console': 'off',
    'no-debugger': 'off',
    'no-empty': 'off',
    'no-ex-assign': 'off',
    'no-extra-boolean-cast': 'off',
    'no-fallthrough': 'off',
    'no-prototype-builtins': 'off',
    'no-undef': 'off',
    'no-unused-vars': 'off',
    'react/display-name': 'off',
    'react/no-direct-mutation-state': 'off',
    'react/prop-types': 'off',
    'react/no-unescaped-entities': 'off',
  },
};
