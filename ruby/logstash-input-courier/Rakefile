require 'rubygems'
require 'rubygems/package_task'

gemspec = Gem::Specification.load('logstash-input-courier.gemspec')
Gem::PackageTask.new(gemspec).define

task :default do
  Rake::Task[:package].invoke
end

task :clean do
  sh 'rm -rf .bundle'
  sh 'rm -rf pkg'
  sh 'rm -rf vendor'
end
