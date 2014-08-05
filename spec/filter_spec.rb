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

describe 'log-courier with filter codec' do
  include_context 'Helpers'
  include_context 'Helpers_Log_Courier'

  it 'should filter events' do
    startup stdin: true, config: <<-config
    {
      "network": {
        "ssl ca": "#{@ssl_cert.path}",
        "servers": [ "127.0.0.1:#{server_port}" ],
        "timeout": 15,
        "reconnect": 1
      },
      "files": [
        {
          "paths": [ "-" ],
          "codec": { "name": "filter", "pattern": "^stdin line test [12]" }
        }
      ]
    }
    config

    5_000.times do |i|
      @log_courier.puts "stdin line test #{i}"
    end

    # Receive and check
    i = 0
    host = Socket.gethostname
    receive_and_check(total: 2_778) do |e|
      expect(e['message']).to eq "stdin line test #{i}"
      expect(e['host']).to eq host
      expect(e['file']).to eq '-'
      i += 1
      i += 1 while /^[12]/ =~ i.to_s
    end
  end
end
