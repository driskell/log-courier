$LOAD_PATH << File.join(File.dirname(File.dirname(File.dirname(__FILE__))), 'lib')

require 'lib/helpers/common'
require 'lib/logfile'

TEMP_PATH = File.join(File.dirname(File.dirname(__FILE__)), 'tmp')
STARTUP_WAIT_TIME = 2
EVENT_WAIT_COUNT = 50
EVENT_WAIT_TIME = 0.5

RSpec.configure do |config|
  config.before :all do
    FileUtils.rm_r(TEMP_PATH) if File.directory?(TEMP_PATH)
    Dir.mkdir(TEMP_PATH)
    Dir.mkdir(File.join(TEMP_PATH, 'logs'))
  end

  config.before :each do
    puts "\n\n"
  end

  config.after :each do
    puts "\n\n"
  end

  config.before :all do
    puts "\n\n"
  end
end
