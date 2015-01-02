Gem::Specification.new do |gem|
  gem.name              = 'logstash-output-log-courier'
  gem.version           = '1.3'
  gem.description       = 'Log Courier Output Logstash Plugin'
  gem.summary           = 'Transmit events from one Logstash instance to another using the Log Courier protocol'
  gem.homepage          = 'https://github.com/driskell/log-courier'
  gem.authors           = ['Jason Woods']
  gem.email             = ['devel@jasonwoods.me.uk']
  gem.licenses          = ['Apache']
  gem.rubyforge_project = 'nowarning'
  gem.require_paths     = ['lib']
  gem.files             = %w(
    lib/logstash/outputs/courier.rb
  )

  gem.metadata = { 'logstash_plugin' => 'true', 'group' => 'input' }

  gem.add_runtime_dependency 'logstash', '~> 1.4'
  gem.add_runtime_dependency 'log-courier', '= 1.3'
end
