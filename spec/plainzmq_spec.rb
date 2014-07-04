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

describe 'log-courier' do
  include_context 'Helpers'
  include_context 'Helpers_Log_Courier'

  before :all do
    @transport = 'plainzmq'
  end

  it 'should allow plain ZMQ transport' do
    # Hide lines in the file - this makes sure we start at the end of the file
    f = create_log

    startup config: <<-config
    {
      "network": {
        "transport": "plainzmq",
        "servers": [ "127.0.0.1:#{server_port}" ]
      },
      "files": [
        {
          "paths": [ "#{TEMP_PATH}/logs/log-*" ]
        }
      ]
    }
    config

    f.log 5_000

    # Receive and check
    receive_and_check
  end
end
