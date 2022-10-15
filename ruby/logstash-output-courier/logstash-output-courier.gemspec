# Add platform conditions around java-only dependencies so GitHub dependency chart that is MRI only (I think) still works
Gem::Specification.new do |gem|
  gem.name              = 'logstash-output-courier'
  gem.version           = '2.9.1'
  gem.description       = 'Courier Output Logstash Plugin'
  gem.summary           = 'Transmit events from one Logstash instance to another using the Courier protocol'
  gem.homepage          = 'https://github.com/driskell/log-courier'
  gem.authors           = ['Jason Woods']
  gem.email             = ['devel@jasonwoods.me.uk']
  gem.platform          = 'java' if RUBY_PLATFORM == 'java'
  gem.licenses          = ['Apache-2.0']
  gem.rubyforge_project = 'nowarning'
  gem.require_paths     = ['lib']
  gem.files             = [
    'lib/logstash/outputs/courier.rb',
  ]

  gem.metadata = { 'logstash_plugin' => 'true', 'logstash_group' => 'output' }

  gem.add_runtime_dependency 'log-courier', '= 2.9.1'
  gem.add_runtime_dependency 'logstash-codec-plain' if RUBY_PLATFORM == 'java'
  gem.add_runtime_dependency 'logstash-core-plugin-api', '>= 1.60', '<= 2.99' if RUBY_PLATFORM == 'java'
end
