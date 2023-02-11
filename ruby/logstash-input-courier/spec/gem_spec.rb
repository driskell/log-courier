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
require 'logstash/inputs/courier'

describe LogStash::Inputs::Courier do
  include_context 'LogCourier'

  context 'logstash-input-courier' do
    it 'receives connections and generates events' do
      @plugin = LogStash::Inputs::Courier.new(
        'port' => 12_345,
        'ssl_certificate' => @ssl_cert.path,
        'ssl_key' => @ssl_key.path,
      )
      @plugin.register

      @thread = Thread.new do
        @plugin.run @event_queue
      end

      client = start_client(
        port: 12_345,
      )

      # Allow 60 seconds
      Timeout.timeout(60) do
        5_000.times do |i|
          client.publish 'message' => "gem line test #{i}", 'host' => @host, 'path' => 'gemfile.log'
        end

        # Receive and check
        i = 0
        receive_and_check(total: 5_000) do |e|
          e = e.to_hash
          expect(e['message']).to eq "gem line test #{i}"
          expect(e['host']).to eq @host
          expect(e['path']).to eq 'gemfile.log'
          i += 1
        end
      end

      shutdown_client
      @plugin.stop
      @thread.join
    end
  end
end
