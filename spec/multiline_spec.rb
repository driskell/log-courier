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

require 'lib/common'
require 'lib/helpers/log-courier'
require 'lib/logfile/multiline'

describe 'log-courier with multiline codec' do
  include_context 'Helpers'
  include_context 'Helpers_Log_Courier'

  it 'should combine multiple events with what=previous' do
    startup config: <<-config
    {
      "network": {
        "ssl ca": "#{@ssl_cert.path}",
        "servers": [ "127.0.0.1:#{server_port}" ],
        "timeout": 15,
        "reconnect": 1
      },
      "files": [
        {
          "paths": [ "#{TEMP_PATH}/logs/log-*" ],
          "codec": { "name": "multiline", "what": "previous", "pattern": "^\\\\s" }
        }
      ]
    }
    config

    f = create_log(LogFile::Multiline)

    5000.times do |i|
      f.log
    end

    # We will always be missing the last line - this is expected behaviour as we cannot know the last multiline block is complete
    f.skip_one

    # Receive and check
    receive_and_check
  end

  it 'should combine multiple events with what=previous and negate' do
    startup config: <<-config
    {
      "network": {
        "ssl ca": "#{@ssl_cert.path}",
        "servers": [ "127.0.0.1:#{server_port}" ],
        "timeout": 15,
        "reconnect": 1
      },
      "files": [
        {
          "paths": [ "#{TEMP_PATH}/logs/log-*" ],
          "codec": { "name": "multiline", "what": "previous", "pattern": "^BEGIN", "negate": true }
        }
      ]
    }
    config

    f = create_log(LogFile::Multiline)

    5_000.times do
      f.log
    end

    # We will always be missing the last line - this is expected behaviour as we cannot know the last multiline block is complete
    f.skip_one

    # Receive and check
    receive_and_check
  end

  it 'should combine multiple events with what=previous and previous_timeout' do
    startup config: <<-config
    {
      "network": {
        "ssl ca": "#{@ssl_cert.path}",
        "servers": [ "127.0.0.1:#{server_port}" ],
        "timeout": 15,
        "reconnect": 1
      },
      "files": [
        {
          "paths": [ "#{TEMP_PATH}/logs/log-*" ],
          "codec": { "name": "multiline", "what": "previous", "pattern": "^\\\\s", "previous timeout": "10s" }
        }
      ]
    }
    config

    f = create_log(LogFile::Multiline)

    5_000.times do
      f.log
    end

    # Receive and check
    receive_and_check
  end

  it 'should combine multiple events with what=next' do
    startup config: <<-config
    {
      "network": {
        "ssl ca": "#{@ssl_cert.path}",
        "servers": [ "127.0.0.1:#{server_port}" ],
        "timeout": 15,
        "reconnect": 1
      },
      "files": [
        {
          "paths": [ "#{TEMP_PATH}/logs/log-*" ],
          "codec": { "name": "multiline", "what": "next", "pattern": "[0-9]$" }
        }
      ]
    }
    config

    f = create_log(LogFile::Multiline)

    5_000.times do
      f.log
    end

    # Receive and check
    receive_and_check
  end

  it 'should combine multiple events with what=next and negate' do
    startup config: <<-config
    {
      "network": {
        "ssl ca": "#{@ssl_cert.path}",
        "servers": [ "127.0.0.1:#{server_port}" ],
        "timeout": 15,
        "reconnect": 1
      },
      "files": [
        {
          "paths": [ "#{TEMP_PATH}/logs/log-*" ],
          "codec": { "name": "multiline", "what": "next", "pattern": "^\\\\sEND", "negate": true }
        }
      ]
    }
    config

    f = create_log(LogFile::Multiline)

    5_000.times do
      f.log
    end

    # Receive and check
    receive_and_check
  end
end
