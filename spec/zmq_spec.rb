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

describe 'log-courier with zmq' do
  include_context 'Helpers'
  include_context 'Helpers_Log_Courier'

  before :all do
    @transport = 'zmq'
  end

  it 'should distribute events to multiple peers' do
    # Start another 4 peers
    start_server id: 'peer2'
    start_server id: 'peer3'
    start_server id: 'peer4'
    start_server id: 'peer5'

    f = create_log

    startup config: <<-config
    {
      "network": {
        "transport": "zmq",
        "curve server key": "i@tV)lm/:sbI-ODWpD[*7kn2[19[DcUBWnZ2)LJ>",
        "curve public key": "6aoJA{jXq[j8y>mTE:&XkW3kUD]8zK&SiVv]KJ?j",
        "curve secret key": "Z8U#fkH%z1e9lJLIuQ=P(mC)8GJQ?sdcGxi*l(5W",
        "servers": [
          "127.0.0.1:#{server_port}",
          "127.0.0.1:#{server_port('peer2')}",
          "127.0.0.1:#{server_port('peer3')}",
          "127.0.0.1:#{server_port('peer4')}",
          "127.0.0.1:#{server_port('peer5')}"
        ],
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

    # Send LOTS of lines but don't overdo it
    # If Ruby gets too busy receiving them we might duplicate a payload and
    # the test will fail
    100_000.times do
      f.log
    end

    # Receive and check
    receive_and_check check_order: false

    # Make sure we received on ALL endpoints
    expect(server_count).to be > 0
    expect(server_count('peer2')).to be > 0
    expect(server_count('peer3')).to be > 0
    expect(server_count('peer4')).to be > 0
    expect(server_count('peer5')).to be > 0
  end

  it 'should distribute events to multiple peers and manage send failures' do
    # Start another 4 peers, 1 of which is TLS so it'll act like a dead endpoint
    start_server id: 'peer2'
    start_server id: 'peer3', transport: 'tls'
    start_server id: 'peer4'
    start_server id: 'peer5'

    f = create_log

    startup config: <<-config
    {
      "network": {
        "transport": "zmq",
        "curve server key": "i@tV)lm/:sbI-ODWpD[*7kn2[19[DcUBWnZ2)LJ>",
        "curve public key": "6aoJA{jXq[j8y>mTE:&XkW3kUD]8zK&SiVv]KJ?j",
        "curve secret key": "Z8U#fkH%z1e9lJLIuQ=P(mC)8GJQ?sdcGxi*l(5W",
        "servers": [
          "127.0.0.1:#{server_port}",
          "127.0.0.1:#{server_port('peer2')}",
          "127.0.0.1:#{server_port('peer3')}",
          "127.0.0.1:#{server_port('peer4')}",
          "127.0.0.1:#{server_port('peer5')}"
        ],
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

    # Send lines - just enough for all 5 endpoints
    6_000.times do
      f.log
    end

    # Receive and check
    receive_and_check check_order: false
  end
end
