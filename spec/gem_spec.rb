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

require 'cabin'
require 'timeout'
require 'lib/common'

require 'log-courier/client'

describe 'log-courier gem' do
  include_context 'Helpers'

  before :all do
    @host = Socket.gethostname
  end

  def startup
    logger = Cabin::Channel.new
    logger.subscribe SHARED_LOGGER_OUTPUT
    logger.level = :debug

    # Reset server for each test
    @client = LogCourier::Client.new(
      :ssl_ca => @ssl_cert.path,
      :addresses => ['127.0.0.1'],
      :port => server_port,
      :logger => logger
    )
  end

  def shutdown
    @client.shutdown
  end

  it 'should send and receive events' do
    startup

    # Allow 60 seconds
    Timeout.timeout(60) do
      5_000.times do |i|
        @client.publish 'message' => "gem line test #{i}", 'host' => @host, 'path' => 'gemfile.log'
      end
    end

    # Receive and check
    i = 0
    receive_and_check(total: 5_000) do |e|
      expect(e['message']).to eq "gem line test #{i}"
      expect(e['host']).to eq @host
      expect(e['path']).to eq 'gemfile.log'
      i += 1
    end

    expect(shutdown).to eq true
  end
end
