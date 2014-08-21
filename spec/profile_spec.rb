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

require 'ruby-prof'
require 'multi_json'

describe 'profile' do
  include_context 'Helpers'
  include_context 'Helpers_Log_Courier'

  PROFILE_LINES = 500_000

  before :all do
    puts ':: Initialising'
  #  @transport = 'zmq'
  end

  after :all do
    puts ':: Done'
  #  @transport = 'zmq'
  end

  it "should process #{PROFILE_LINES} events and produce a profiler result" do
    # Start another 4 peers
    #start_server id: 'peer2'
    #start_server id: 'peer3'
    #start_server id: 'peer4'
    #start_server id: 'peer5'

    # Nice big log file
    puts ":: Preparing #{PROFILE_LINES} lines in a log file"
    f = create_log
    c = 0
    BENCHMARK_LINES.times do
      c += 1
      f.log
      puts ":: #{c} lines written" if c % (PROFILE_LINES / 10) == 0
    end

    puts ":: Beginning profile using multi_json engine #{MultiJson::adapter}"
    RubyProf.start

    startup verbose: false, args: '-from-beginning=true', config: <<-config
    {
      "general": {
        "admin enabled": true,
        "admin port": 1234
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
    #  "general": {
    #    "admin enabled": true,
    #    "admin port": 1234
    #  },
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
    profile = RubyProf.stop
    puts ":: Received #{BENCHMARK_LINES} lines"

    # Print a flat profile to text
    puts ":: Profile results:"
    printer = RubyProf::FlatPrinter.new(profile)
    printer.print(STDOUT)
  end
end
