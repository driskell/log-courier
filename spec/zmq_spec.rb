require "lib/common"
require "lib/helpers/lsf"

describe "logstash-forwarder with zmq" do
  include_context "Helpers"
  include_context "Helpers_LSF"

  it "should distribute events to multiple peers" do
    # Start another 4 peers
    start_server 'peer2'
    start_server 'peer3'
    start_server 'peer4'
    start_server 'peer5'

    f = create_log()

    startup config: <<-config
    {
      "network": {
        "servers": [
          "127.0.0.1:#{server_port()}",
          "127.0.0.1:#{server_port('peer2')}",
          "127.0.0.1:#{server_port('peer3')}",
          "127.0.0.1:#{server_port('peer4')}",
          "127.0.0.1:#{server_port('peer5')}"
        ],
        "transport": {
          "name": "zmq"
        },
        "timeout": 15,
        "reconnect": 1
      },
      "files": [
        {
          "paths": [ "#{TEMP_PATH}/logs/log-*" ]
        }
      ]
    }
    config

    # Send LOTS of lines
    500000.times do |i|
      f.log
    end

    # Receive and check
    receive_and_check
  end
end
