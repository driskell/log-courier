Gem::Specification.new do |gem|
  gem.name              = 'log-courier'
  gem.version           = '1.10'
  gem.description       = 'Log Courier library'
  gem.summary           = 'Ruby implementation of the Courier protocol'
  gem.homepage          = 'https://github.com/driskell/ruby-log-courier'
  gem.authors           = ['Jason Woods']
  gem.email             = ['devel@jasonwoods.me.uk']
  gem.licenses          = ['Apache']
  gem.rubyforge_project = 'nowarning'
  gem.require_paths     = ['lib']
  gem.files             = %w(
    lib/log-courier/client.rb
    lib/log-courier/client_tcp.rb
    lib/log-courier/event_queue.rb
    lib/log-courier/server.rb
    lib/log-courier/server_tcp.rb
    lib/log-courier/server_zmq.rb
    lib/log-courier/zmq_qpoll.rb
  )

  gem.add_runtime_dependency 'cabin',      '~> 0.6'
  gem.add_runtime_dependency 'ffi-rzmq',   '~> 2.0'
  gem.add_runtime_dependency 'multi_json', '~> 1.10'
end
