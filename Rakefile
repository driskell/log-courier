task :default do
  system('rake -T')
end

# Publish gem to rubygems
# https://github.com/logstash-plugins/logstash-input-tcp/blob/master/rakelib/publish.rake
desc 'Publish gem to RubyGems.org'
task :publish_gem do
  require 'gem_publisher'
  gem_file = Dir.glob(File.expand_path('*.gemspec', File.dirname(__FILE__))).first
  gem = GemPublisher.publish_if_updated(gem_file, :rubygems)
  puts "Published #{gem}" if gem
end
