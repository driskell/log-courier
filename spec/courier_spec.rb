# encoding: utf-8

# Copyright 2014-2015 Jason Woods.
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

require 'rbconfig'
require 'lib/common'
require 'lib/helpers/log-courier'

describe 'log-courier' do
  include_context 'Helpers'
  include_context 'Helpers_Log_Courier'

  it 'should follow stdin' do
    startup stdin: true, config: <<-config
    {
      "general": {
        "persist directory": "."
      },
      "network": {
        "ssl ca": "#{@ssl_cert.path}",
        "servers": [ "localhost:#{server_port}" ]
      },
      "stdin": {
        "fields": { "type": "stdin" }
      }
    }
    config

    # Remember the sized queue we use for test buffering is only 10_000 lines
    5_000.times do |i|
      @log_courier.puts "stdin line test #{i}"
    end

    # Receive and check
    i = 0
    host = Socket.gethostname
    receive_and_check(total: 5_000) do |e|
      expect(e['message']).to eq "stdin line test #{i}"
      expect(e['host']).to eq host
      expect(e['path']).to eq 'stdin'
      expect(e['type']).to eq 'stdin'
      i += 1
    end

    stdin_shutdown
  end

  it 'should split lines that are too long' do
    startup stdin: true, config: <<-config
    {
      "general": {
        "persist directory": "."
      },
      "network": {
        "ssl ca": "#{@ssl_cert.path}",
        "servers": [ "127.0.0.1:#{server_port}" ]
      }
    }
    config

    # This should send over the 10 MiB packet limit but not break it
    # Since we are sending 10 * 2 = 20 MiB
    10.times do |i|
      # 1048575 since the line terminater adds 1 - ensures second part fits
      @log_courier.puts 'X' * 1_048_575 * 2
    end

    # Receive and check
    i = 0
    host = Socket.gethostname
    receive_and_check(total: 20) do |e|
      if i.even?
        expect(e['message'].length).to eq 1_048_576
        expect(e['tags']).to eq ['splitline']
      else
        expect(e['message'].length).to eq 1_048_574
        expect(e.has_key?('tags')).to eq false
      end
      expect(e['host']).to eq host
      expect(e['path']).to eq 'stdin'
      i += 1
    end

    stdin_shutdown
  end

  it 'should follow a file from the end' do
    # Hide lines in the file - this makes sure we start at the end of the file
    f = create_log.log(50).skip

    startup

    f.log 5_000

    # Receive and check
    receive_and_check
  end

  it 'should follow a file from the beginning with parameter -from-beginning=true' do
    # Hide lines in the file - this makes sure we start at the beginning of the file
    f = create_log.log(50)

    startup args: '-from-beginning=true'

    f.log 5000

    # Receive and check
    receive_and_check
  end

  it 'should follow a slowly-updating file' do
    startup

    f = create_log

    100.times do |i|
      f.log 50

      # Start fast, then go slower after 80% of the events
      # Total sleep becomes 20 seconds
      sleep 1 if i > 80
    end

    # Quickly test we received at least 90% already
    # If not, then the 5 second idle_timeout has been ignored and test fails
    expect(@event_queue.length).to be >= 4_500

    # Receive and check
    receive_and_check
  end

  it 'should follow multiple files and resume them when restarted' do
    f1 = create_log
    f2 = create_log

    startup

    5000.times do
      f1.log
      f2.log
    end

    # Receive and check
    receive_and_check

    # Now restart logstash
    shutdown

    # From beginning makes testing this easier - without it we'd need to create lines inbetween shutdown and start and verify them which is more work
    startup args: '-from-beginning=true'

    5_000.times do
      f1.log
      f2.log
    end

    # Receive and check
    receive_and_check
  end

  it 'should start newly created files found after startup from beginning and not the end' do
    # Create a file and hide it
    f1 = create_log.log(5_000)
    path = f1.path
    f1.rename File.join(File.dirname(path), 'hide-' + File.basename(path))

    startup

    create_log.log 5_000

    # Throw the file back with all the content already there
    # We can't just create a new one, it might pick it up before we write
    f1.rename path

    # Receive and check
    receive_and_check
  end

  it 'should handle log rotation and resume correctly' do
    f1 = create_log

    startup

    f1.log 100

    # Receive and check
    receive_and_check

    # Rotate f1 - this renames it and returns a new file same name as original f1
    f2 = rotate(f1)

    # Write to both
    f1.log 5_000
    f2.log 5_000

    # Receive and check
    receive_and_check

    # Restart
    shutdown
    startup

    # Write some more
    f1.log 5_000
    f2.log 5_000

    # Receive and check - but not file as it will be different now
    receive_and_check check_file: false
  end

  it 'should handle log rotation and resume correctly with symlinked log files', :unless => RbConfig::CONFIG['host_os'] =~ /mswin|mingw|cygwin/ do
    config = <<-config
    {
      "general": {
        "persist directory": "."
      },
      "network": {
        "ssl ca": "#{@ssl_cert.path}",
        "servers": [ "127.0.0.1:#{server_port}" ]
      },
      "files": [
        {
          "paths": [ "#{TEMP_PATH}/logs/log" ],
          "dead time": "15s"
        }
      ]
    }
    config

    startup config: config

    f1 = create_log.log 100
    File.symlink f1.path, "#{TEMP_PATH}/logs/log"

    # Receive and check, but do not check file due to symlink
    receive_and_check check_file: false

    4.times do
      f2 = rotate(f1)
      f1 = f2
      f2.log 1_024
      sleep 10
      f1.log 1_024
      receive_and_check check_file: false
    end

    # Restart
    shutdown
    startup config: config, args: '-from-beginning=true'

    # Write some more
    f1.log 1_000

    # Receive and check
    receive_and_check check_file: false
  end

  it 'should handle log rotation during startup resume' do
    startup

    f1 = create_log.log 100

    # Receive and check
    receive_and_check

    # Stop
    shutdown

    # Rotate f1 - this renames it and returns a new file same name as original f1
    f2 = rotate(f1)

    # Write to both
    f1.log 5_000
    f2.log 5_000

    # Start again
    startup

    # Receive and check - but not file as it will be different now
    receive_and_check check_file: false
  end

  it 'should resume harvesting a file that reached dead time but changed again' do
    startup config: <<-config
    {
      "general": {
        "persist directory": "."
      },
      "network": {
        "ssl ca": "#{@ssl_cert.path}",
        "servers": [ "127.0.0.1:#{server_port}" ]
      },
      "files": [
        {
          "paths": [ "#{TEMP_PATH}/logs/log-*" ],
          "dead time": "5s"
        }
      ]
    }
    config

    f1 = create_log.log(5_000)

    # Receive and check
    receive_and_check

    # Let dead time occur
    sleep 15

    # Write again
    f1.log(5_000)

    # Receive and check
    receive_and_check
  end

  it 'should prune deleted files from registrar state' do
    # We use dead time to make sure the harvester stops, as file deletion is only acted upon once the harvester stops
    startup config: <<-config
    {
      "general": {
        "persist directory": "."
      },
      "network": {
        "ssl ca": "#{@ssl_cert.path}",
        "servers": [ "127.0.0.1:#{server_port}" ]
      },
      "files": [
        {
          "paths": [ "#{TEMP_PATH}/logs/log-*" ],
          "dead time": "5s"
        }
      ]
    }
    config

    # Write lines
    f1 = create_log.log(5_000)
    create_log.log 5_000

    # Receive and check
    receive_and_check

    # Grab size of the saved state - sleep to ensure it was saved
    sleep 1
    s = File::Stat.new('.log-courier').size

    # Close and delete one of the files
    f1.close

    # Wait for prospector to realise it is deleted
    sleep 15

    # Check new size of registrar state
    expect(File::Stat.new('.log-courier').size).to be < s
  end

  it 'should response to SIGHUP by reloading configuration', :unless => RbConfig::CONFIG['host_os'] =~ /mswin|mingw|cygwin/ do
    start_server id: 'new'

    f1 = create_log
    f2 = create_log

    startup config: <<-config
    {
      "general": {
        "persist directory": "."
      },
      "network": {
        "ssl ca": "#{@ssl_cert.path}",
        "servers": [ "127.0.0.1:#{server_port}" ]
      },
      "files": [
        {
          "paths": [ "#{TEMP_PATH}/logs/log-0" ]
        }
      ]
    }
    config

    # Write logs
    f1.log 5_000

    # Receive and check
    receive_and_check

    # Extra lines for the next file
    f2.log 5_000

    # Reload configuration
    reload <<-config
    {
      "general": {
        "persist directory": "."
      },
      "network": {
        "ssl ca": "#{@ssl_cert.path}",
        "servers": [ "127.0.0.1:#{server_port('new')}" ]
      },
      "files": [
        {
          "paths": [ "#{TEMP_PATH}/logs/log-0" ]
        },
        {
          "paths": [ "#{TEMP_PATH}/logs/log-1" ]
        }
      ]
    }
    config

    # Receive and check
    receive_and_check

    expect(server_count('new')).to be > 0
  end

  it 'should allow use of a custom persist directory' do
    f = create_log

    config = <<-config
    {
      "general": {
        "persist directory": "#{TEMP_PATH}"
      },
      "network": {
        "ssl ca": "#{@ssl_cert.path}",
        "servers": [ "127.0.0.1:#{server_port}" ]
      },
      "files": [
        {
          "paths": [ "#{TEMP_PATH}/logs/log-*" ]
        }
      ]
    }
    config

    startup config: config

    # Write logs
    f.log 5_000

    # Receive and check
    receive_and_check

    # Restart - use from-beginning so we fail if we don't resume
    shutdown
    startup config: config, args: '-from-beginning=true'

    # Write some more
    f.log 5_000

    # Receive and check
    receive_and_check

    # We have to clean up ourselves here since .log-courer is elsewhere
    # Do some checks to ensure we used a different location though
    shutdown
    expect(File.file?(".log-courier")).to be false
    expect(File.file?("#{TEMP_PATH}/.log-courier")).to be true
    File.unlink("#{TEMP_PATH}/.log-courier")
  end

  it 'should allow multiple fields to be configured' do
    startup config: <<-config
    {
      "general": {
        "persist directory": "."
      },
      "network": {
        "ssl ca": "#{@ssl_cert.path}",
        "servers": [ "127.0.0.1:#{server_port}" ]
      },
      "files": [
        {
          "paths": [ "#{TEMP_PATH}/logs/log-0" ],
          "fields": { "first": "value", "second": "more" }
        },
        {
          "paths": [ "#{TEMP_PATH}/logs/log-1" ],
          "fields": { "first": "different", "second": "something" }
        }
      ]
    }
    config

    f1 = create_log.log 5_000
    f2 = create_log.log 5_000

    # Receive and check
    receive_and_check(total: 10_000) do |e|
      if e['path'] == "#{TEMP_PATH}/logs/log-0"
        expect(e['first']).to eq "value"
        expect(e['second']).to eq "more"
      else
        expect(e['path']).to eq "#{TEMP_PATH}/logs/log-1"
        expect(e['first']).to eq "different"
        expect(e['second']).to eq "something"
      end
    end
  end

  it 'should allow arrays inside field configuration' do
    startup stdin: true, config: <<-config
    {
      "general": {
        "persist directory": "."
      },
      "network": {
        "ssl ca": "#{@ssl_cert.path}",
        "servers": [ "127.0.0.1:#{server_port}" ]
      },
      "stdin": {
        "fields": { "array": [ 1, 2 ] }
      }
    }
    config

    5_000.times do |i|
      @log_courier.puts "stdin line test #{i}"
    end

    # Receive and check
    i = 0
    host = Socket.gethostname
    receive_and_check(total: 5_000) do |e|
      expect(e['message']).to eq "stdin line test #{i}"
      expect(e['array']).to be_kind_of Array
      expect(e['array'][0]).to eq 1
      expect(e['array'][1]).to eq 2
      expect(e['host']).to eq host
      expect(e['path']).to eq 'stdin'
      i += 1
    end

    stdin_shutdown
  end

  it 'should allow dictionaries inside field configuration' do
    startup stdin: true, config: <<-config
    {
      "general": {
        "persist directory": "."
      },
      "network": {
        "ssl ca": "#{@ssl_cert.path}",
        "servers": [ "127.0.0.1:#{server_port}" ]
      },
      "stdin": {
        "fields": { "dict": { "first": "first", "second": 5 } }
      }
    }
    config

    5_000.times do |i|
      @log_courier.puts "stdin line test #{i}"
    end

    # Receive and check
    i = 0
    host = Socket.gethostname
    receive_and_check(total: 5_000) do |e|
      expect(e['message']).to eq "stdin line test #{i}"
      expect(e['dict']).to be_kind_of Hash
      expect(e['dict']['first']).to eq 'first'
      expect(e['dict']['second']).to eq 5
      expect(e['host']).to eq host
      expect(e['path']).to eq 'stdin'
      i += 1
    end

    stdin_shutdown
  end

  it 'should accept globs of configuration files to include' do
    # Create a set of files
    f1 = create_log
    f2 = create_log
    f3 = create_log

    config = <<-config
    {
      "general": {
        "persist directory": "."
      },
      "network": {
        "ssl ca": "#{@ssl_cert.path}",
        "servers": [ "127.0.0.1:#{server_port}" ]
      },
      "files": [
        {
          "paths": [ "#{TEMP_PATH}/logs/log-0" ]
        }
      ],
      "includes": [ "#{TEMP_PATH}/include-*" ]
    }
    config

    includes = []

    begin
      [1,2].each do |i|
        include_file = File.open(File.join(TEMP_PATH, 'include-' + i.to_s + '.json'), 'w')
        includes.push include_file.path
        include_file.puts <<-config
        [
          {
            "paths": [ "#{TEMP_PATH}/logs/log-#{i.to_s}" ]
          }
        ]
        config
        include_file.close
      end

      startup config: config

      f1.log 5_000
      f2.log 5_000
      f3.log 5_000

      # Receive and check
      receive_and_check
    ensure
      includes.each do |include_file|
        File.unlink(include_file) if File.file?(include_file)
      end
    end
  end

  # TODO: We should start using Go tests for things like this
  it 'should accept the various general and network configuration elements' do
    f = create_log

    startup config: <<-config
    {
      "general": {
        "persist directory": ".",
        "prospect interval": 10,
        "spool size": 1024,
        "spool timeout": 5,
        "log level": "debug",
        "log syslog": false,
        "log stdout": false,
        "log file": "#{TEMP_PATH}/logs/log.log",
        "host": "custom.hostname.local"
      },
      "network": {
        "ssl ca": "#{@ssl_cert.path}",
        "servers": [ "127.0.0.1:#{server_port}" ],
        "timeout": 15,
        "failure backoff": 1,
        "failure backoff max": 60,
        "reconnect backoff": 1,
        "reconnect backoff max": 60
      },
      "files": [
        {
          "paths": [ "#{TEMP_PATH}/logs/log-*" ]
        }
      ]
    }
    config

    f.log 5000

    # Receive and check
    receive_and_check host: 'custom.hostname.local'

    shutdown
    expect(File.file?("#{TEMP_PATH}/logs/log.log")).to be true
    File.unlink "#{TEMP_PATH}/logs/log.log" if File.file? "#{TEMP_PATH}/logs/log.log"
  end
end
