require 'rubygems'
require 'rubygems/package_task'

gemspec = Gem::Specification.load('logstash-output-courier.gemspec')
Gem::PackageTask.new(gemspec).define

task :default => [:deploy] do
  require 'rspec/core/rake_task'
  RSpec::Core::RakeTask.new(:spec)
  Rake::Task[:spec].invoke
end

task :deploy do
  Bundler.with_clean_env do
    sh 'bundle install --deployment'
  end
end

task :update do
  Bundler.with_clean_env do
    sh 'bundle install --no-deployment --path vendor/bundle'
  end
end

task :release => [:package] do
  sh "gem push pkg/logstash-output-courier-#{gemspec.version}.gem"
end

task :clean do
  sh 'rm -rf .bundle'
  sh 'rm -rf pkg'
  sh 'rm -rf vendor'
end
