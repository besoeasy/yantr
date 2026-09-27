import js from '@eslint/js'
import pluginVue from 'eslint-plugin-vue'
import pluginA11y from 'eslint-plugin-vuejs-accessibility'
import globals from 'globals'

/**
 * ESLint flat config.
 *
 * ui/ previously had no lint step at all (only dev/build/preview), which is how
 * a `div` with a click handler shipped as the only way to open a stack's
 * service panel — a keyboard-only user could not read stats, read logs, or reach
 * the container page from that view at all. The a11y rules below exist so that
 * specific class of regression fails instead of waiting to be reported.
 *
 * Deliberately a CORRECTNESS lint, not a style lint. Running
 * `pluginVue.configs['flat/recommended']` produces ~2,150 warnings on this
 * codebase (html-indent, max-attributes-per-line, attributes-order, …) because
 * it enforces a formatting convention the project has never adopted. A linter
 * that reports two thousand warnings gets ignored, which defeats the purpose, so
 * only the base parser config is used and correctness rules are opted into by
 * hand. Run `npm run lint:fix` for the autofixable subset.
 */

// Config files run in Node, not the browser.
const nodeFiles = ['*.config.js', 'eslint.config.js']

export default [
  { ignores: ['dist/**', 'node_modules/**'] },

  js.configs.recommended,
  // Parser + minimal Vue wiring only. No stylistic rule sets.
  ...pluginVue.configs['flat/base'],
  ...pluginA11y.configs['flat/recommended'],
  { linterOptions: { reportUnusedDisableDirectives: true } },

  {
    files: ['**/*.js', '**/*.vue'],
    languageOptions: {
      ecmaVersion: 2023,
      sourceType: 'module',
      globals: { ...globals.browser, ...globals.es2021 },
    },
    plugins: { vue: pluginVue, 'vuejs-accessibility': pluginA11y },
    rules: {
      // ── The rules that would have caught the #98 blocker ──
      // A non-interactive element with a click handler, and one with a click
      // handler but no keyboard equivalent. Both as errors.
      'vuejs-accessibility/no-static-element-interactions': 'error',
      'vuejs-accessibility/click-events-have-key-events': 'error',
      'vuejs-accessibility/mouse-events-have-key-events': 'error',
      // A role="button" div with no tabindex is still unreachable by keyboard.
      'vuejs-accessibility/interactive-supports-focus': 'error',
      // A control with no programmatic name. The env-var and port inputs were
      // siblings of their <label> with no `for`, and several icon-only buttons
      // were announced as just "button".
      'vuejs-accessibility/form-control-has-label': 'error',
      'vuejs-accessibility/label-has-for': [
        'error',
        // `some` rather than the rule's default `every: ['nesting', 'id']`.
        // The default additionally requires the control to be *nested* inside
        // the label, which flags the equally valid sibling-plus-`for` pattern
        // this codebase uses throughout. `some` still rejects a label that has
        // neither a nested control nor a matching id.
        { required: { some: ['nesting', 'id'] } },
      ],
      // An anchor with no text and no icon child is announced as just "link".
      // Note this does NOT catch `<a href="#">`, which was the actual defect in
      // Volumes (three of them) — the plugin has no href-validity rule, so those
      // were fixed by hand and are not guarded here.
      'vuejs-accessibility/anchor-has-content': 'error',

      // ── Vue correctness ──
      // Single-word SFCs are the established convention here (App.vue, Logs.vue,
      // Home.vue); renaming them would be churn for no benefit.
      'vue/multi-word-component-names': 'off',
      // An undefined component reference is a real bug; template-key duplication
      // and a missing key are not worth failing a build over in this codebase.
      'vue/no-unused-components': 'warn',
      'vue/require-explicit-emits': 'off',
      'vue/no-v-html': 'off',
      'vue/attributes-order': 'off',
      'vue/html-self-closing': 'off',
      'vue/html-indent': 'off',
      'vue/html-closing-bracket-newline': 'off',
      'vue/first-attribute-linebreak': 'off',
      'vue/max-attributes-per-line': 'off',
      'vue/singleline-html-element-content-newline': 'off',
      'vue/multiline-html-element-content-newline': 'off',
      'vue/attribute-hyphenation': 'off',

      // ── General correctness ──
      // An empty catch block is how a failing request became invisible — the
      // stats panel kept rendering stale numbers with no signal at all.
      'no-empty': ['error', { allowEmptyCatch: false }],
      // Unused vars are how the dead `expirationInfo` computed survived (#97).
      // Vue SFC compiler macros are capitalised, hence the ignore pattern.
      'no-unused-vars': ['error', { args: 'none', varsIgnorePattern: '^[A-Z_]' }],
      'vue/no-unused-vars': 'error',
      'no-console': ['warn', { allow: ['warn', 'error'] }],
      eqeqeq: ['warn', 'smart'],
      'prefer-const': 'error',
      'no-var': 'error',
    },
  },

  {
    files: nodeFiles,
    languageOptions: { globals: { ...globals.node } },
  },
]
