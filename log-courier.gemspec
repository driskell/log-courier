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

# Write the version.rb file
version_rb = IO.read 'lib/log-courier/version.rb.tmpl'
version_rb.gsub!('<VERSION>', version)
IO.write 'lib/log-courier/version.rb', version_rb

Gem::Specification.new do |gem|
  gem.name              = 'log-courier'
  gem.version           = version
  gem.description       = 'Log Courier library'
  gem.summary           = 'Ruby implementation of the Log Courier protocol'
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
