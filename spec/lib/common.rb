# encoding: utf-8

# Copyright 2014 Jason Woods.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

$LOAD_PATH << File.join(File.dirname(File.dirname(File.dirname(__FILE__))), 'lib')

require 'cabin'
require 'lib/helpers/common'
require 'lib/logfile'

TEMP_PATH = File.join(File.dirname(File.dirname(__FILE__)), 'tmp')
STARTUP_WAIT_TIME = 2
EVENT_WAIT_COUNT = 50
EVENT_WAIT_TIME = 0.5
SHARED_LOGGER_OUTPUT = Cabin::Outputs::IO.new(STDOUT)

RSpec.configure do |config|
  config.before :all do
    FileUtils.rm_r(TEMP_PATH) if File.directory?(TEMP_PATH)
    Dir.mkdir(TEMP_PATH)
    Dir.mkdir(File.join(TEMP_PATH, 'logs'))
    puts "\n\n"
  end

  config.after :all do
    FileUtils.rm_r(TEMP_PATH) if File.directory?(TEMP_PATH)
  end

  config.before :each do
    puts "\n\n"
  end

  config.after :each do
    puts "\n\n"
  end
end
