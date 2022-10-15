module.exports = {
  infile: 'CHANGELOG.md',
  preset: {
    name: 'conventionalcommits'
  },
  bumpFiles: [
    'package.json',
    {
      filename: 'lc-lib/core/version.go',
      updater: 'contrib/version-updaters/version.go.js'
    },
    {
      filename: 'ruby/log-courier/lib/log-courier/version.rb',
      updater: 'contrib/version-updaters/version.rb.js'
    },
    {
      filename: 'ruby/log-courier/log-courier.gemspec',
      updater: 'contrib/version-updaters/gemspec.js'
    },
    {
      filename: 'ruby/logstash-input-courier/logstash-input-courier.gemspec',
      updater: 'contrib/version-updaters/gemspec.js'
    },
    {
      filename: 'ruby/logstash-output-courier/logstash-output-courier.gemspec',
      updater: 'contrib/version-updaters/gemspec.js'
    }
  ]
};
