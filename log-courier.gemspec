Gem::Specification.new do |gem|
  gem.name              = 'log-courier'
  gem.version           = '0.9'
  gem.description       = 'Log Courier library'
  gem.summary           = 'An enhanced Logstash Forwarder'
  gem.homepage          = 'https://github.com/driskell/log-courier'
  gem.authors           = ['Jason Woods']
  gem.email             = ['devel@jasonwoods.me.uk']
  gem.rubyforge_project = 'nowarning'
  gem.require_paths     = ['lib']
  gem.files             = %w(
    lib/log-courier/server.rb
    lib/log-courier/server_tls.rb
    lib/log-courier/server_zmq.rb
    lib/log-courier/client.rb
    lib/log-courier/client_tls.rb
  )

  gem.add_runtime_dependency 'ffi-rzmq'
end
