import js from '@eslint/js'
import pluginVue from 'eslint-plugin-vue'
import * as parserVue from 'vue-eslint-parser'
import configPrettier from 'eslint-config-prettier'
import pluginTs from '@typescript-eslint/eslint-plugin'
import parserTs from '@typescript-eslint/parser'

export default [
  js.configs.recommended,
  ...pluginVue.configs['flat/recommended'],
  configPrettier,
  {
    files: ['**/*.ts', '**/*.tsx', '**/*.vue'],
    languageOptions: {
      parser: parserVue,
      parserOptions: {
        parser: parserTs,
        extraFileExtensions: ['.vue'],
        sourceType: 'module',
        ecmaVersion: 'latest'
      },
      globals: {
        window: 'readonly',
        document: 'readonly',
        console: 'readonly'
      }
    },
    plugins: {
      '@typescript-eslint': pluginTs
    },
    rules: {
      '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_' }],
      'vue/multi-word-component-names': 'off',
      '@typescript-eslint/no-explicit-any': 'warn'
    }
  },
  {
    ignores: ['node_modules', 'dist', '*.config.js', '*.config.ts']
  }
]

