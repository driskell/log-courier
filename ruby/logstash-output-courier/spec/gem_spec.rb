# Copyright 2014-2021 Jason Woods.
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

require 'logstash/devutils/rspec/spec_helper'
require 'log-courier/rspec/spec_helper'
require 'logstash/outputs/courier'

describe LogStash::Outputs::Courier do
  include_context 'LogCourier'

  before :all do
    LogStash::Logging::Logger.configure_logging 'debug'
  end

  context 'logstash-output-courier' do
    it 'sends events' do
      @plugin = LogStash::Outputs::Courier.new(
        'hosts' => ['127.0.0.1'],
        'port' => 12_345,
        'ssl_ca' => @ssl_cert.path,
      )
      @plugin.register

      start_server(
        port: 12_345,
      )

      # Allow 60 seconds
      Timeout.timeout(60) do
        5_000.times do |i|
          @plugin.receive 'message' => "gem line test #{i}", 'host' => @host, 'path' => 'gemfile.log'
        end

        # Receive and check
        i = 0
        receive_and_check(total: 5_000) do |e|
          expect(e['message']).to eq "gem line test #{i}"
          expect(e['host']).to eq @host
          expect(e['path']).to eq 'gemfile.log'
          i += 1
        end
      end

      @plugin.close
      shutdown_server
    end
  end
end
