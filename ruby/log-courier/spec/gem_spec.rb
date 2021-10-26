# Copyright 2014-2021 Jason Woods
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

require 'timeout'
require 'log-courier/rspec/spec_helper'

describe 'log-courier gem' do
  include_context 'LogCourier'

  it 'should send and receive events with tcp client' do
    start_server(
      transport: 'tcp',
    )

    client = start_client(
      transport: 'tcp',
    )

    # Allow 60 seconds
    Timeout.timeout(60) do
      5_000.times do |i|
        client.publish 'message' => "gem line test #{i}", 'host' => 'testing.example.com', 'path' => 'gemfile.log'
      end

      # Receive and check
      i = 0
      receive_and_check(total: 5_000) do |e|
        expect(e['message']).to eq "gem line test #{i}"
        expect(e['host']).to eq 'testing.example.com'
        expect(e['path']).to eq 'gemfile.log'
        i += 1
      end
    end

    shutdown_client
    shutdown_server
  end

  it 'should send and receive events with tls client' do
    start_server

    client = start_client

    # Allow 60 seconds
    Timeout.timeout(60) do
      5_000.times do |i|
        client.publish 'message' => "gem line test #{i}", 'host' => 'testing.example.com', 'path' => 'gemfile.log'
      end

      # Receive and check
      i = 0
      receive_and_check(total: 5_000) do |e|
        expect(e['message']).to eq "gem line test #{i}"
        expect(e['host']).to eq 'testing.example.com'
        expect(e['path']).to eq 'gemfile.log'
        i += 1
      end
    end

    shutdown_client
    shutdown_server
  end

  it 'should send and receive events with tls client that does not handshake' do
    start_server

    client = start_client(
      disable_handshake: true,
    )

    # Allow 60 seconds
    Timeout.timeout(60) do
      5_000.times do |i|
        client.publish 'message' => "gem line test #{i}", 'host' => 'testing.example.com', 'path' => 'gemfile.log'
      end

      # Receive and check
      i = 0
      receive_and_check(total: 5_000) do |e|
        expect(e['message']).to eq "gem line test #{i}"
        expect(e['host']).to eq 'testing.example.com'
        expect(e['path']).to eq 'gemfile.log'
        i += 1
      end
    end

    shutdown_client
    shutdown_server
  end

  it 'should send and receive events with tls server that does not handshake' do
    start_server(
      disable_handshake: true,
    )

    client = start_client

    # Allow 60 seconds
    Timeout.timeout(60) do
      5_000.times do |i|
        client.publish 'message' => "gem line test #{i}", 'host' => 'testing.example.com', 'path' => 'gemfile.log'
      end

      # Receive and check
      i = 0
      receive_and_check(total: 5_000) do |e|
        expect(e['message']).to eq "gem line test #{i}"
        expect(e['host']).to eq 'testing.example.com'
        expect(e['path']).to eq 'gemfile.log'
        i += 1
      end
    end

    shutdown_client
    shutdown_server
  end
end
