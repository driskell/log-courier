Gem::Specification.new do |gem|
  gem.authors       = ['Jason Woods']
  gem.email         = ['devel@jasonwoods.me.uk']
  gem.description   = 'Log Courier library'
  gem.summary       = gem.description
  gem.homepage      = 'https://github.com/driskell/log-courier'

  gem.files = %w{
    (lib/log-courier/server.rb)
    (lib/log-courier/server_tls.rb)
    (lib/log-courier/server_zmq.rb)
    (lib/log-courier/client.rb)
    (lib/log-courier/client_tls.rb)
  }

  gem.name          = 'log-courier'
  gem.require_paths = ['lib']
  gem.version       = '0.9'

  gem.add_runtime_dependency 'ffi-rzmq'
end
