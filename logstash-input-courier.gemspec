Gem::Specification.new do |gem|
  gem.name              = 'logstash-input-courier'
  gem.version           = '1.9.1'
  gem.description       = 'Courier Input Logstash Plugin'
  gem.summary           =
    'Receive events from Log Courier and Logstash using the Courier protocol'
  gem.homepage          = 'https://github.com/driskell/logstash-input-courier'
  gem.authors           = ['Jason Woods']
  gem.email             = ['devel@jasonwoods.me.uk']
  gem.licenses          = ['Apache-2.0']
  gem.rubyforge_project = 'nowarning'
  gem.require_paths     = ['lib']
  gem.files             = %w(
    lib/logstash/inputs/courier.rb
  )

  gem.metadata = { 'logstash_plugin' => 'true', 'logstash_group' => 'input' }

  gem.add_runtime_dependency 'logstash-core-plugin-api', '>= 1.60', '<= 2.99'
  gem.add_runtime_dependency 'log-courier', '~> 1.9'

  gem.add_development_dependency 'logstash-codec-plain', '~> 3.0'
  gem.add_development_dependency 'logstash-devutils', '~> 1.2'
end
