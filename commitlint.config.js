module.exports = {
  extends: ['@commitlint/config-conventional'],
  rules: {
    // Disable subject-case
    'subject-case': [
      0,
      'never',
      ['sentence-case', 'start-case', 'pascal-case', 'upper-case'],
    ],
    // Dependabot and others exceed body and body line length so just disable it
    'body-max-length': [0, 'always', 100],
    'body-max-line-length': [0, 'always', 100],
    // Allow longer header length
    'header-max-length': [2, 'always', 150]
  }
};
