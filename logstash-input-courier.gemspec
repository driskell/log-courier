# Pull version from git if we're cloned (git command sure to exist)
# Otherwise, if in an archive, use version.txt, which is the last stable version
if File.directory? '.git'
  version = \
    `git describe | sed 's/-\([0-9][0-9]*\)-\([0-9a-z][0-9a-z]*\)$/-\1.\2/g'`
  version.sub!(/^v/, '')
else
  version = IO.read 'version.txt'
end

version.chomp!

Gem::Specification.new do |gem|
  gem.name              = 'logstash-input-courier'
  gem.version           = version
  gem.description       = 'Courier Input Logstash Plugin'
  gem.summary           = 'Receive events from Log Courier and Logstash using the Courier protocol'
  gem.homepage          = 'https://github.com/driskell/logstash-input-courier'
  gem.authors           = ['Jason Woods']
  gem.email             = ['devel@jasonwoods.me.uk']
  gem.licenses          = ['Apache']
  gem.rubyforge_project = 'nowarning'
  gem.require_paths     = ['lib']
  gem.files             = %w(
    lib/logstash/inputs/courier.rb
  )

  gem.metadata = { 'logstash_plugin' => 'true', 'logstash_group' => 'input' }

  gem.add_runtime_dependency 'logstash-core', '>= 1.4', '< 3'
  gem.add_runtime_dependency 'log-courier', '~> 1.9'
end
