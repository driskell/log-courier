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

task :docs do
  sh 'npm --version >/dev/null' do |ok|
    next if ok
    fail %('npm' not found. You need to install node.js.)
  end
  sh 'npm install doctoc >/dev/null' do |ok|
    next if ok
    fail 'Failed to perform local install of doctoc.'
  end
  sh 'node_modules/.bin/doctoc README.md'
  Rake::FileList['docs/*.md', 'docs/codecs/*.md'].each do |file|
    sh 'node_modules/.bin/doctoc ' + file
  end
end

task :clean do
  sh 'rm -rf .bundle'
  sh 'rm -rf vendor'
end
