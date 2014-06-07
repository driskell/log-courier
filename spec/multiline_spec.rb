require "lib/common"
require "lib/helpers/lsf"
require "lib/logfile/multiline"

describe "logstash-forwarder with multiline codec" do
  include_context "Helpers"
  include_context "Helpers_LSF"

  it "should combine multiple events with what=previous" do
    startup config: <<-config
    {
      "network": {
        "servers": [ "127.0.0.1:#{server_port()}" ],
        "transport": {
          "name": "tls",
          "ssl ca": "#{@ssl_cert.path}"
        },
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

  it "should combine multiple events with what=previous and negate" do
    startup config: <<-config
    {
      "network": {
        "servers": [ "127.0.0.1:#{server_port()}" ],
        "transport": {
          "name": "tls",
          "ssl ca": "#{@ssl_cert.path}"
        },
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

    5000.times do |i|
      f.log
    end

    # We will always be missing the last line - this is expected behaviour as we cannot know the last multiline block is complete
    f.skip_one

    # Receive and check
    receive_and_check
  end

  it "should combine multiple events with what=previous and previous_timeout" do
    startup config: <<-config
    {
      "network": {
        "servers": [ "127.0.0.1:#{server_port()}" ],
        "transport": {
          "name": "tls",
          "ssl ca": "#{@ssl_cert.path}"
        },
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

    5000.times do |i|
      f.log
    end

    # Receive and check
    receive_and_check
  end

  it "should combine multiple events with what=next" do
    startup config: <<-config
    {
      "network": {
        "servers": [ "127.0.0.1:#{server_port()}" ],
        "transport": {
          "name": "tls",
          "ssl ca": "#{@ssl_cert.path}"
        },
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

    5000.times do |i|
      f.log
    end

    # Receive and check
    receive_and_check
  end

  it "should combine multiple events with what=next and negate" do
    startup config: <<-config
    {
      "network": {
        "servers": [ "127.0.0.1:#{server_port()}" ],
        "transport": {
          "name": "tls",
          "ssl ca": "#{@ssl_cert.path}"
        },
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

    5000.times do |i|
      f.log
    end

    # Receive and check
    receive_and_check
  end
end
