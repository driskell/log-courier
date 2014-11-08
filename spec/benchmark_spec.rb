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

require 'multi_json'

describe 'benchmark' do
  include_context 'Helpers'
  include_context 'Helpers_Log_Courier'

  BENCHMARK_LINES = 500_000

  before :all do
    puts ':: Initialising'
  #  @transport = 'zmq'
  end

  after :all do
    puts ':: Done'
  #  @transport = 'zmq'
  end

  it "should process #{BENCHMARK_LINES} events as quickly as possible" do
    # Start another 4 peers
    #start_server id: 'peer2'
    #start_server id: 'peer3'
    #start_server id: 'peer4'
    #start_server id: 'peer5'

    # Nice big log file
    puts ":: Preparing #{BENCHMARK_LINES} lines in a log file"
    f = create_log
    c = 0
    BENCHMARK_LINES.times do
      c += 1
      f.log
      puts ":: #{c} lines written" if c % (BENCHMARK_LINES / 10) == 0
    end

    puts ":: Beginning timed benchmark using multi_json engine #{MultiJson::adapter}"
    start_time = Time.now

    startup verbose: false, args: '-from-beginning=true', config: <<-config
    {
      "general": {
        "admin enabled": true,
        "admin listen address": "tcp:127.0.0.1:12350",
        "log level": "debug"
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

    #startup args: '-from-beginning=true', config: <<-config
    #{
    #  "network": {
    #    "transport": "zmq",
    #    "curve server key": "i@tV)lm/:sbI-ODWpD[*7kn2[19[DcUBWnZ2)LJ>",
    #    "curve public key": "6aoJA{jXq[j8y>mTE:&XkW3kUD]8zK&SiVv]KJ?j",
    #    "curve secret key": "Z8U#fkH%z1e9lJLIuQ=P(mC)8GJQ?sdcGxi*l(5W",
    #    "servers": [
    #      "127.0.0.1:#{server_port}",
    #      "127.0.0.1:#{server_port('peer2')}",
    #      "127.0.0.1:#{server_port('peer3')}",
    #      "127.0.0.1:#{server_port('peer4')}",
    #      "127.0.0.1:#{server_port('peer5')}"
    #    ]
    #  },
    #  "files": [
    #    {
    #      "paths": [ "#{TEMP_PATH}/logs/log-*" ]
    #    }
    #  ]
    #}
    #config

    # Receive and check
    receive_and_check check: false

    # Make sure we received on ALL endpoints
    #expect(server_count).to be > 0
    #expect(server_count('peer2')).to be > 0
    #expect(server_count('peer3')).to be > 0
    #expect(server_count('peer4')).to be > 0
    #expect(server_count('peer5')).to be > 0

    # Shutdown
    shutdown

    # Output time
    elapsed_time = ((Time.now - start_time) * 1000.0).to_i
    puts ":: Received #{BENCHMARK_LINES} lines in #{elapsed_time} ms"
    puts ":: That's #{BENCHMARK_LINES * 1000 / elapsed_time} per second"
  end
end
