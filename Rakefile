require 'rubygems'

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

task :clean do
  sh 'rm -rf .bundle'
  sh 'rm -rf vendor'
end
